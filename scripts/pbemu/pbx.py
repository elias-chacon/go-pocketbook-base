#!/usr/bin/env python3
"""pbx: drive the pbemu emulator (screenshot, touch, swipe, launch, install, logs).

  pbx.py shot OUT.png            current screen as PNG
  pbx.py tap X Y                 touch
  pbx.py swipe X1 Y1 X2 Y2 [N] [MS]   drag in N steps, MS ms apart
  pbx.py install FILE.app        copy into /mnt/ext1/applications
  pbx.py launch NAME             start /mnt/ext1/applications/NAME.app as a task
  pbx.py tasks                   list the tasks
  pbx.py sh CMD...               shell in the container
"""
import os, subprocess, sys, time, zlib, struct

ROOT = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.join(ROOT, "pbemu", "tools"))
FW = os.environ.get("PB_FW", "U741_6.8.4079")
DK = os.environ.get("PODMAN", "docker")
CT = "pb-pocketbook-ui"
LIVE = os.path.join(ROOT, "pbemu", FW, ".live")
APPS = os.path.join(LIVE, "mnt", "ext1", "applications")


def sh(cmd, timeout=60):
    return subprocess.run([DK, "exec", CT, "sh", "-lc", cmd], capture_output=True, text=True, timeout=timeout)


def probe(name, *args):
    return subprocess.run([DK, "exec", CT, f"/workspace/src/viewer/build-pc/{name}", *args],
                          capture_output=True, text=True, timeout=30)


def png(path, w, h, rows):
    raw = b"".join(b"\0" + r for r in rows)
    def chunk(t, d):
        c = struct.pack(">I", len(d)) + t + d
        return c + struct.pack(">I", zlib.crc32(t + d) & 0xFFFFFFFF)
    with open(path, "wb") as f:
        f.write(b"\x89PNG\r\n\x1a\n" + chunk(b"IHDR", struct.pack(">IIBBBBB", w, h, 8, 2, 0, 0, 0))
                + chunk(b"IDAT", zlib.compress(raw, 3)) + chunk(b"IEND", b""))


def shot(out):
    r = probe("fb_pgm", "/tmp/pbx.ppm")
    if r.returncode:
        sys.exit(r.stderr)
    b = open(os.path.join(LIVE, "tmp", "pbx.ppm"), "rb").read()
    hdr = b.split(b"\n", 3)
    m, (w, h) = hdr[0], map(int, hdr[1].split())
    data = hdr[3]
    c = 3 if m == b"P6" else 1
    rows = []
    for y in range(h):
        row = data[y * w * c:(y + 1) * w * c]
        if c == 3:
            ba = bytearray(row)
            ba[0::3], ba[2::3] = row[2::3], row[0::3]  # BGR -> RGB
            row = bytes(ba)
        else:
            row = bytes(v for p in row for v in (p, p, p))
        rows.append(row)
    png(out, w, h, rows)
    print(r.stdout.strip())


def ev(*a):
    r = probe("send_event", *map(str, a))
    if r.returncode:
        sys.exit(r.stderr)


def main():
    a = sys.argv[1:]
    if not a:
        sys.exit(__doc__)
    c = a[0]
    if c == "shot":
        shot(a[1])
    elif c == "tap":
        ev("touch", a[1], a[2])
    elif c == "swipe":
        x1, y1, x2, y2 = map(int, a[1:5])
        n = int(a[5]) if len(a) > 5 else 8
        ms = int(a[6]) if len(a) > 6 else 30
        ev("down", x1, y1)
        for i in range(1, n + 1):
            time.sleep(ms / 1000)
            ev("move", x1 + (x2 - x1) * i // n, y1 + (y2 - y1) * i // n)
        ev("up", x2, y2)
    elif c == "install":
        os.makedirs(APPS, exist_ok=True)
        dst = os.path.join(APPS, os.path.basename(a[1]))
        subprocess.run(["cp", a[1], dst], check=True)
        os.chmod(dst, 0o755)
        print("installed", dst)
    elif c == "launch":
        path = f"/mnt/ext1/applications/{a[1]}.app"
        cmd = ("exec qemu-arm -L /workspace/firmware/.live/guest "
               + qemu_env() + " /workspace/src/probes/arm/build-arm/arm_probe launch " + " ".join(a[2:]) + " -- " + path)
        r = sh(cmd)
        print(r.stdout, r.stderr)
    elif c == "run":
        # run an app directly (not as a task of the launcher) keeping its stderr in /tmp/NAME.out
        cmd = ("exec qemu-arm -L /workspace/firmware/.live/guest " + qemu_env()
               + f" /mnt/ext1/applications/{a[1]}.app > /tmp/{a[1]}.out 2>&1 < /dev/null &")
        sh(cmd)
        print("started", a[1])
    elif c == "tasks":
        r = sh("exec qemu-arm -L /workspace/firmware/.live/guest " + qemu_env()
               + " /workspace/src/probes/arm/build-arm/arm_probe task-list")
        print(r.stdout, r.stderr)
    elif c == "sh":
        r = sh(" ".join(a[1:]))
        print(r.stdout, r.stderr)
    else:
        sys.exit(__doc__)


def qemu_env():
    from pbemu.run import qemu_env_args
    return qemu_env_args()


if __name__ == "__main__":
    main()
