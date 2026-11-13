# 7. Events and state

[← Components](06-components.md) · [Index](README.md) · [Next: Recipes →](08-recipes.md)

## Callbacks at a glance

All callbacks have the same shape and are plain fields, like `onclick` in JavaScript:

```go
type Handler func(e *widget.Event)

type Event struct {
    Target widget.Widget // the element the event is about
    X, Y   float64       // touch position in design pixels (click events)
}
func (e *Event) StopPropagation()
```

| Callback | Elements | When |
|---|---|---|
| `OnClick` | every element | the element was tapped (**bubbles** to the ancestors) |
| `OnChange` | check-box/radio/switch, combo-box, inputs, text area, slider, chip, table (`OnSelect`), calendar | the value changed by the user |
| `OnInput` | slider | while dragging |
| `OnSubmit` | search input | magnifier tapped or OK on the keyboard |
| `OnSelect` | dropdown, table, `Menu`, `MenuItem` | an item or row was chosen |
| `OnClose` | chip | the × was tapped |

## Reading values

The element is also the model: read its properties inside the callback (or any time).

| Element | Read |
|---|---|
| input, text area | `Value`; for numbers `Number()`, for masks the digits in `Value` and the text in `Display()` |
| check-box, radio, switch | `Checked` (radio: `Value` is the option's value) |
| combo-box | `Value()`, `Index` |
| slider | `Value` |
| chips | `chip.Selected`, `group.SelectedLabels()` |
| table | `Selected` (index in `Rows`), `Rows[Selected]` |
| calendar | `Selected` (`time.Time`, zero when none) |

`e.Target` is the element, so one handler can serve many elements:

```go
for _, r := range []*widget.Toggle{free, pro} {
	r.OnChange = func(e *widget.Event) {
		fmt.Println("plan:", e.Target.(*widget.Toggle).Value)
	}
}
```

## Changing the screen

Change properties and the next frame shows them. **A tap already requests a repaint**, so inside callbacks you
just assign:

```go
label.Text = "Saved"          // text
errorLabel.Hidden = false     // show / hide
button.Disabled = true        // grey out
slider.SetValue(70)           // move the knob
table.Rows = newRows          // new data
table.Page, table.Selected = 0, -1
```

Outside a callback (for example after your own code finished a job) call `doc.Invalidate()` — or
`element.Invalidate()` — to ask for a repaint.

### Adding and removing elements

```go
list.AppendChild(widget.NewChip("new"))   // add at the end
list.RemoveChild(chip)                    // remove one
list.ClearChildren()                      // remove all (useful to swap a page)
```

Swapping pages is just that: clear the container and append the other page. The
[navigation recipe](08-recipes.md#navigation-between-pages-with-the-menu-bar) does exactly this.

### Finding elements

```go
doc.GetElementByID("save")           // like document.getElementById
doc.QuerySelectorAll("danger")       // every element with ClassName "danger"
doc.Walk(func(w widget.Widget) bool { /* visit all */ return true })
```

Set `ID` and `ClassName` on the elements you need to find later.

## Stopping the bubble

A tap on a button inside a card runs the button's `OnClick`, then the card's, then the page's. Stop it when
the parent must not react:

```go
button.OnClick = func(e *widget.Event) {
	e.StopPropagation() // the card's OnClick will not run
}
```

## Disabled and hidden

- `Disabled = true`: drawn washed out, **receives no touch**, and neither do its descendants.
- `Hidden = true`: not drawn, takes no space, **receives no touch**.

## Popups and the keyboard

- Combo-boxes, dropdowns and menu bars open a **popup** above everything. A tap outside only closes it.
  `doc.HasPopup()` tells whether one is open and `doc.ClosePopups()` closes it (the Back key does this).
- Input fields call `doc.RequestText(...)` which uses `doc.Keyboard`. `doc.Focused()` returns the field being
  edited, or nil.
- On the reader the **Back** key arrives through `inkview` (`KeyBack`); the app glue decides what it does.

## Advanced: raw touch

You normally never need this, but a custom element can receive the three touch phases:

```go
func (t *TapCounter) HandlePointer(e *widget.PointerEvent) {
	// e.Type is PointerDown, PointerMove, PointerUp or PointerClick
}
```

See [Custom components](09-custom-components.md).

Next: [Recipes →](08-recipes.md)
