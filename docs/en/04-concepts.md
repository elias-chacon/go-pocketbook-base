# 4. Concepts

[← Build and versioning](03-build-and-versioning.md) · [Index](README.md) · [Next: Layout →](05-layout.md)

This page explains how the pieces fit together, so the rest of the documentation makes sense.

## The layers

```
your screens (examples/, cmd/…)          what you write
        │
internal/widget                          elements, layout, touch dispatch
        │
internal/gfx  ·  internal/geom           vector drawing, geometry
        │
internal/inkview                         the reader: events, screen, keyboard (cgo)
```

`widget` and `gfx` are **pure Go**: they run on your PC, which is why the whole UI can be tested and
previewed without a device. Only `inkview` needs the reader (on the desktop it has stubs that do nothing).

## Document, Widget and Base

- A **`Document`** owns one tree of elements. It lays the tree out, draws it into an image and routes touch.
  Create it with `widget.NewDocument(screenWidth, screenHeight)` and give it a root with `SetRoot`.
- A **`Widget`** is any element. The interface is tiny:

  ```go
  type Widget interface {
      Elem() *Base                    // the common properties
      Measure(avail float64) geom.Size // natural content size (leaf elements)
      Paint(c *gfx.Canvas)             // draws the element's own content
  }
  ```
- **`Base`** is embedded by every element and gives it the properties of a web element:

  | Property / method | Meaning |
  |---|---|
  | `ID`, `ClassName` | identification, like `id` and `class` |
  | `Style` | layout and look (see [Layout](05-layout.md)) |
  | `Hidden`, `Disabled` | hidden elements vanish; disabled ones are greyed out and ignore touch |
  | `OnClick` | callback for taps (bubbles up to the ancestors) |
  | `Bounds` | where the element is on screen, set by the layout |
  | `AppendChild`, `RemoveChild`, `ClearChildren`, `Children`, `Parent` | the tree |
  | `Document()`, `Invalidate()` | the owner document; ask for a repaint |

  Elements nest: a `Card` contains a `Label`, an `Input` and a `Button`, a `Row` contains cards… The
  tree has one root.

## Coordinates: design pixels

All sizes are written in **design pixels**, where the screen is **1404 px wide** (the PB741). On a
screen with another resolution the document computes a scale (`device width / 1404`) and converts
everything, so the same code looks the same on any reader:

```go
doc := widget.NewDocument(758, 1024) // a smaller reader: Scale = 0.54, design height = 1896
```

- `doc.W` and `doc.H` are the design size (use them to size things);
- `doc.Scale` converts to real pixels;
- touch coordinates arrive in real pixels and are converted back, so **drawing and touch always use the
  same space**: what you see is exactly where you touch.

Sizes in the library follow the e-ink: controls are at least **84 px** tall (about 9 mm on the PB741) and
the smallest touch area is **64 px**.

## From code to pixels

Every time the screen must change, the document runs three steps (`doc.Render()`):

1. **Layout** — each element receives its rectangle (`Bounds`) from the flex/grid engine.
2. **Paint** — the tree is drawn depth-first: first the box of each element (background, border) from its
   `Style`, then its own content (`Paint`), then its children. The open modal comes next, then open popups, on top.
3. **Gray image** — the result is an 8-bit grayscale image; on the reader it is copied to the screen and
   refreshed.

```go
img := doc.Render() // *image.Gray, device-sized; save it as a PNG or blit it to the e-ink
```

The look is grayscale on purpose: high contrast, crisp borders, no animation. The colors are
`widget.Ink` (black), `Paper` (white), `Soft`, `Mid`, `Muted` and `Disabled` (grays).

## Repainting only what changed

A full paint takes about a second on the reader, but a tap usually changes one small thing, so the glue calls
`doc.Update()` instead of `doc.Render()`:

1. it lays the document out and takes a **snapshot** of every element (its fields and its bounds);
2. it compares the snapshot with the previous one: every element that differs (or appeared, or disappeared)
   marks its old and new bounds as **dirty**;
3. it repaints only the dirty areas, into a small canvas, and patches them into the last frame;
4. it returns a `Frame`: the complete image plus the list of changed rectangles, so only those are sent to the panel.

Your code does not announce anything: setting `cb.Checked = true` or `label.Text = "..."` in a handler is enough,
exactly as with a full repaint. The result matches a full `Render()` (a test taps every element of every page and
compares), so this is purely an optimization.

The whole screen is repainted when more than 40% of it changed (a page switch), after the system keyboard, when the
system asks for a redraw on its own, and by the glue every 10 partial updates (a full e-ink refresh clears ghosting).
`doc.ResetFrame()` forces it.

Things to know when you write your own elements:

- Everything that changes how an element looks must be in the element's **fields** (or in `Document` state like
  focus). State kept behind a pointer is invisible to the snapshot: implement `StateSignature() string` to describe it
  (`MenuBar` does).
- An element may paint at most 8 design px outside its `Bounds`.
- The snapshot prints every field, so a table with thousands of rows makes each update slower.

## From touch to callbacks

Touch goes through `doc.HandlePointer(phase, x, y)` with three phases: **down**, **move** and **up**.

1. On **down**, the document finds the deepest element under the finger that reacts to touch (an element
   with a `HandlePointer` method or an `OnClick`). A **disabled or hidden** element never gets it.
2. **Move** and **up** go to that same element (this is how a slider can be dragged).
3. If the finger is released over the element it started on, a **click** is produced: the element runs its
   own logic (a check-box flips, a combo-box opens…), then `OnClick` runs on the element and on each
   ancestor up to the root (**bubbling**), until someone calls `e.StopPropagation()`.
4. The document asks for a repaint.

Popups (combo-box lists, menus) are drawn above everything. While one is open, a touch **outside** it
only closes it, and never reaches the element underneath.

A **modal** (`widget.Modal`) sits between the page and the popups: while it is open (and not minimized) it takes every touch,
and a touch outside it does nothing. Popups opened from inside the modal are drawn above it.

## Typing: the system keyboard

Input fields do not draw a keyboard. When tapped they ask the `Keyboard` of the document to open the
reader's own on-screen keyboard:

```go
type Keyboard interface {
    Request(widget.KeyboardRequest) // Title, Initial, Kind (text/numeric), MaxLen, Done(text, ok)
}
```

On the reader the app passes the InkView keyboard; in tests and previews you pass a `FakeKeyboard`
that answers immediately. A field being edited is *focused* (it gets a thicker border) until the
keyboard closes.

## The app on the reader

`cmd/hello` and `examples/newapp` are a thin glue between InkView and the document:

| InkView event | What the glue does |
|---|---|
| start | turn the system panel off, read the screen size, build the document |
| show | `doc.Update()` and send only the changed areas to the screen (`examples/newapp`: `doc.Render()` and the whole image) |
| touch down / move / up | `doc.HandlePointer(...)` |
| key (Back, or left / Menu on the PB741) | `doc.HandleBack()`: close a popup, then a modal |
| key (previous / next page) | the demo changes tab (`MovePage`) |

Two details worth knowing:

- **Repaints are coalesced.** `doc.OnInvalidate` is called for every change, but the glue queues at most
  one repaint at a time, and skips intermediate *move* events while the screen catches up. That keeps
  dragging a slider smooth on e-ink.
- **Change the screen from event handlers.** Callbacks run on the reader's main thread, which is the
  safe place to modify elements. This base has no timer or background-task API yet.

## Why it is built this way

- **Small interfaces** (`Widget`, `PointerHandler`, `Keyboard`) so a screen never depends on the hardware.
- **One layout engine** for everything; each component only says how big it wants to be and how to draw.
- **Elements as plain objects** with web-like names, ready for a future HTML/CSS front end.
- **Tests first-class**: touch accuracy and layout are checked automatically.

Next: [Layout →](05-layout.md)
