package widget

import (
	"go-pocketbook-base/internal/geom"
	"go-pocketbook-base/internal/gfx"
)

// Widget is implemented by every element. Embed Base to get the common
// DOM-like behavior and override Measure/Paint as needed.
type Widget interface {
	// Elem returns the common properties.
	Elem() *Base
	// Measure returns the natural content size of a leaf element given at
	// most avail pixels of horizontal content space. Containers don't need it.
	Measure(avail float64) geom.Size
	// Paint draws the element's own content (the background and border from
	// Style are already drawn). Children are painted afterwards.
	Paint(c *gfx.Canvas)
}

// Base holds what every element has, named after the DOM.
type Base struct {
	ID        string
	ClassName string
	Style     Style
	Hidden    bool
	Disabled  bool
	OnClick   Handler

	// Bounds is the border box in design pixels, set by the layout.
	Bounds geom.Rect

	self     Widget
	parent   Widget
	children []Widget
	doc      *Document
}

// Init links the embedded Base to its outer widget. Every constructor of a
// custom element must call it (e.g. b.Init(b)) before the element is used.
func (b *Base) Init(self Widget) { b.self = self }

// Elem returns b itself (satisfies Widget).
func (b *Base) Elem() *Base { return b }

// Measure has no natural size by default (containers size from their children).
func (b *Base) Measure(float64) geom.Size { return geom.Size{} }

// Paint draws nothing by default.
func (b *Base) Paint(*gfx.Canvas) {
	// Intentionally empty: containers are only their Style box and children.
}

// Self returns the outer widget (e.g. the *Button that embeds this Base).
func (b *Base) Self() Widget { return b.self }

// Parent returns the parent element, or nil.
func (b *Base) Parent() Widget { return b.parent }

// Children returns the child elements.
func (b *Base) Children() []Widget { return b.children }

// Document returns the document the element is attached to, or nil.
func (b *Base) Document() *Document { return b.doc }

// Visible reports whether the element takes part in layout, painting and touch.
func (b *Base) Visible() bool { return !b.Hidden && b.Style.Display != DisplayNone }

// ContentRect is the border box minus border and padding.
func (b *Base) ContentRect() geom.Rect { return b.Bounds.Inset(b.Style.boxInsets()) }

// AppendChild adds child as the last child (moving it if it had a parent).
func (b *Base) AppendChild(child Widget) {
	cb := child.Elem()
	if cb.parent != nil {
		cb.parent.Elem().RemoveChild(child)
	}
	cb.parent = b.self
	b.children = append(b.children, child)
	if b.doc != nil {
		attach(child, b.doc)
	}
	b.Invalidate()
}

// RemoveChild detaches child if it is a child of b.
func (b *Base) RemoveChild(child Widget) {
	for i, c := range b.children {
		if c == child {
			b.children = append(b.children[:i:i], b.children[i+1:]...)
			child.Elem().parent = nil
			attach(child, nil)
			b.Invalidate()
			return
		}
	}
}

// ClearChildren removes all children.
func (b *Base) ClearChildren() {
	for _, c := range b.children {
		c.Elem().parent = nil
		attach(c, nil)
	}
	b.children = nil
	b.Invalidate()
}

// Invalidate asks the document to repaint.
func (b *Base) Invalidate() {
	if b.doc != nil {
		b.doc.Invalidate()
	}
}

// attach sets (or clears) the document of w and all its descendants.
func attach(w Widget, d *Document) {
	b := w.Elem()
	b.doc = d
	for _, c := range b.children {
		attach(c, d)
	}
}
