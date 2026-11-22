# 25. The firmware emulator and the Engineering app

Two resources to test without the device for every idea: the **`pbemu` emulator**, which runs the real firmware of the PB741 on the PC,
and the **Engineering** app (`apps/eng`, context `internal/engineering`), a menu of tests of the screen, the colors, the touch and the
speed that writes everything to a log.

> **What was verified and what was not.** The emulator runs the firmware 6.8.4079 of the PB741 for real (the `libinkview`, the home
> screen, the `monitor`), and the apps of this base run on it: they register their icon, show up in Apps, open, draw in 24 bits and
> answer to touch. But the emulator **has no display panel**: the ioctl that sends an update to the screen is ignored, so it **does not
> show how A2 looks, how the colors of the Kaleido panel look, or how long a real refresh takes**. Those numbers only come from the
> device, with the Engineering app.

## The emulator (`scripts/pbemu.ps1`)

[pbemu](https://codeberg.org/datyoma/pbemu) is a user-mode emulator: the ARM processes of the firmware (`monitor.app`, the apps) run
under `qemu-arm` in a container, and a `LD_PRELOAD` shim fakes the hardware (framebuffer, RTC, touch input). The firmware is
PocketBook's official one, **downloaded on your machine and never committed** (its licence is theirs).

```powershell
.\scripts\pbemu.ps1 setup                     # once: clones pbemu, downloads the PB741 firmware (378 MB), builds the image
.\scripts\pbemu.ps1 start                     # turns the emulated reader on
.\build.ps1 -App hello                        # builds the .app for ARM
.\scripts\pbemu.ps1 install hello             # copies it into the emulated reader
.\scripts\pbemu.ps1 stop; .\scripts\pbemu.ps1 start   # the home screen only rereads the app list when it starts
.\scripts\pbemu.ps1 tap 1200 1560             # Apps; then touch the icon of the app
.\scripts\pbemu.ps1 shot screen.png           # the screen, as a PNG (1404 x 1872)
.\scripts\pbemu.ps1 swipe 700 1200 700 400    # drags a finger
.\scripts\pbemu.ps1 log eng                   # the log files of the app on the emulated reader
.\scripts\pbemu.ps1 crash                     # the last Go `panic` that `monitor.log` holds
```

Requirements: Windows with WSL2 and Docker inside WSL (the emulator runs without a window: the screen comes out through `shot`). `setup`
uses the commit of pbemu that was tested (`scripts/pbemu/setup.sh`), translates the `podman` calls of pbemu into `docker`
(`scripts/pbemu/podman-shim.sh`) and builds a probe, `fb_pgm`, that writes the framebuffer. WSL shuts its VM down when no session is
open and takes the container with it, so `start` leaves a hidden `sleep` session running.

What it is good for: checking the icon registration, the lifecycle, color drawing, touch, intents between apps, the disk (the
`applications/.pbgo` folder of the emulated reader is in `~/pbemu-work/pbemu/U741_6.8.4079/.live/mnt/ext1`) and **finding crashes that
only show up with the real `libinkview`** (that is how an index overflow in the color drawing was found). What it is not good for: the
speed and the look of the e-ink screen.

## The Engineering app

It shows up in the Launcher and in Apps as **Engineering** (`eng.app`, like every app of the Release). It **does nothing by itself**: each
menu item is a test, and every number measured goes to `applications/.pbgo/eng.log`, **one JSON line per event** (`t`, `kind` and the
fields), to be read over USB and analysed later.

| Item | What it does | What it records |
|---|---|---|
| **Colors** | 8 full-screen pictures: color bars with 16 grays, primaries, 16 gray levels, a 256-gray ramp, hues, saturation, white, black | `pattern`: the mode, the time to generate, to draw and for the screen to settle |
| **Screen** | plain fills, 1 px and 8 px checkers, 1 px lines, a 100 px grid with edges, diagonals and corners (dead pixels, ghosting, geometry) | `pattern` |
| **Touch** | an area that draws the path of the finger and shows events, events per second, the longest pause and the pixels | `touch` (phase, x, y and the time since the last event) and `touch_stroke` (the summary of the stroke) |
| **Update modes** | redraws the lower half of the screen in each InkView mode (`partial`, `dynamic`, `a2`, `du4`, `bw`, `hq`, `full`) and measures the call, the time until the screen is done and until it settles; *Samples by mode* shows pictures in each mode to compare by eye | `bench` (each call) and `bench_mode` (mean, p95, min, max, total and settle) |
| **Scrolling** | a list of 300 rows that is dragged with a finger; the *Mode* button chooses how the screen is refreshed while it scrolls and *Auto-scroll* makes 40 steps of 160 px timing each one | `scroll` (each change by touch, with the interval) and `scroll_step`/`scroll_run` (mean, p95, frames per second) |
| **System** | model, hardware depth, the waveform file, the canvas, the screen, the version, free space and the log folder | `system` |
| **Log** | the last lines of the log and a button to clear it | |

Touch a full-screen picture: the top left corner goes back, the top right corner changes the update mode, the left third goes to the
previous picture and the rest to the next one. The caption at the bottom left says which picture and which mode. The left and Menu keys
of the device also go back. The log starts again when it passes 4 MB, and every event reaches the file at once (except `touch` and
`scroll`, which wait for the end of the test so as not to slow what is measured), so what was measured before a crash is not lost.

### What the app gives the platform

- `platform.Env.Panel` (and `platform.Panel`): the panel beyond the platform's own repaint. It draws a **gray or color** image (24 bits,
  byte order B, G, R, confirmed against the PB741 firmware in the emulator) and refreshes a region with an InkView mode
  (`UpdatePartial`, `UpdateDynamic`, `UpdateA2`, `UpdateDU4`, `UpdateBW`, `UpdateHQ`, `UpdateFull`), waits for the panel to finish (`Wait`),
  leaves A2 with a clean update (`Settle`) and chooses the mode of the platform's own repaints (`SetMode`).
- `platform.Raw`: an app that sometimes paints the screen itself (the pictures of the Engineering app) implements `Raw(p Panel) bool`;
  when it returns `true` the platform does not paint over it.
- The drawing calls refuse a rectangle outside the picture instead of reading past the end of the buffer.

### How to analyse the log

```sh
# the mean time of each update mode
grep bench_mode eng.log
# the frames per second of the scroll in each mode
grep scroll_run eng.log
# the touch event rate
grep touch_stroke eng.log
```

Every line is JSON: `jq` works, or any JSON Lines reader.

### Limits

- Nothing was measured on the device yet: the numbers the emulator produces (for instance ~44 ms per update, the same in every mode)
  come from the emulated CPU and **are worthless**.
- A2, and updating over a color layer, were not seen on an e-ink screen; the app exists precisely for that.
- What the PocketBook SDK's `ScrollView` does (A2 while dragging, `DitherAreaPattern2Level`, `ExitUpdateA2` on release) was read in
  the disassembly of `libinkview.so`; the toolkit's `Scroll` does **not** use A2 yet.
