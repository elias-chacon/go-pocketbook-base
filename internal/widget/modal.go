package widget

import (
	"go-pocketbook-base/internal/geom"
	"go-pocketbook-base/internal/gfx"
)

// ModalButtons selects the buttons in a modal's footer.
type ModalButtons int

// Modal footers.
const (
	ModalOK        ModalButtons = iota // one OK button (the default)
	ModalOKCancel                      // Cancel and OK
	ModalNoButtons                     // no footer
)

// ModalControls are the window buttons in a modal's title bar (combine with |).
type ModalControls int

// Modal title bar buttons; the zero value shows none.
const (
	ModalClose ModalControls = 1 << iota
	ModalMaximize
	ModalMinimize
)

// ModalState is how much of the screen a modal takes.
type ModalState int

// Modal states.
const (
	ModalNormal    ModalState = iota // centered, as tall as its content
	ModalMaximized                   // fullscreen
	ModalMinimized                   // only the title bar, docked at the bottom; the page stays usable
)

// Modal layout, in design pixels.
const (
	modalMargin     = 48.0  // minimum gap to the screen edges
	modalDefaultW   = 900.0 // width unless Style.Width is set
	modalButtonMinW = 180.0
)

// Modal is a dialog drawn above the page: a title bar (with optional close,
// maximize and minimize buttons), a body for any content and a footer with OK,
// Cancel or nothing. While open it takes every touch, except when minimized.
//
//	m := widget.NewModal("About", widget.NewLabel("Hello"))
//	m.Open(doc) // OK closes it; Back closes it too (as Cancel)
//
// The Modal is a Panel, so it can also be placed inline in a layout.
type Modal struct {
	Panel
	Title string
	// Body holds the content; add to it or style it (e.g. Body.Style.Gap).
	Body *Panel
	// OKText and CancelText label the footer buttons (call Refresh after changing).
	OKText, CancelText string
	// OnOK runs when OK is tapped, OnCancel for Cancel, the close button and the
	// Back key. Either way the modal closes afterwards.
	OnOK, OnCancel Handler

	buttons  ModalButtons
	controls ModalControls
	state    ModalState
}

// NewModal creates a modal with an OK button and no window buttons.
func NewModal(title string, content ...Widget) *Modal {
	m := &Modal{Title: title, OKText: "OK", CancelText: "Cancel"}
	m.Init(m)
	s := &m.Style
	s.Display, s.FlexDirection = DisplayFlex, Column
	s.Background, s.BorderColor, s.BorderWidth, s.BorderRadius = Paper, Ink, 4, Radius+4
	m.Body = NewColumn(18, content...)
	m.Body.Style.Padding = geom.All(28)
	m.Body.Style.FlexGrow = 1 // so the footer stays at the bottom when maximized
	m.Refresh()
	return m
}

// SetButtons chooses the footer buttons.
func (m *Modal) SetButtons(b ModalButtons) { m.buttons = b; m.Refresh() }

// SetControls chooses the title bar buttons, e.g. ModalClose|ModalMaximize.
func (m *Modal) SetControls(c ModalControls) { m.controls = c; m.Refresh() }

// State is the current size state.
func (m *Modal) State() ModalState { return m.state }

// SetState maximizes, minimizes or restores the modal (it may be called before Open).
func (m *Modal) SetState(s ModalState) {
	if m.state == s {
		return
	}
	m.state = s
	m.Refresh()
	if m.doc != nil {
		m.doc.ClosePopups() // their anchors moved
	}
	m.Invalidate()
}

// Open shows the modal above the page of doc.
func (m *Modal) Open(doc *Document) { doc.OpenModal(m) }

// Close hides the modal without running OnOK or OnCancel.
func (m *Modal) Close() {
	if m.doc != nil {
		m.doc.CloseModal(m)
	}
}

// Confirm runs OnOK and closes (what the OK button does).
func (m *Modal) Confirm() {
	Fire(m.OnOK, m)
	m.Close()
}

// Cancel runs OnCancel and closes (the Cancel and close buttons, and Back).
func (m *Modal) Cancel() {
	Fire(m.OnCancel, m)
	m.Close()
}

// blocks reports whether the modal takes the touches that fall outside it.
func (m *Modal) blocks() bool { return m.state != ModalMinimized }

// Refresh rebuilds the title bar and footer from the current options.
func (m *Modal) Refresh() {
	m.ClearChildren()
	m.AppendChild(m.titleBar())
	m.Body.Hidden = m.state == ModalMinimized
	if m.Body.Hidden {
		return
	}
	m.AppendChild(separator())
	m.AppendChild(m.Body)
	if m.buttons != ModalNoButtons {
		m.AppendChild(separator())
		m.AppendChild(m.footer())
	}
}

func separator() *Panel {
	p := NewPanel()
	p.Style.Height, p.Style.Background = Px(2), Mid
	return p
}

func (m *Modal) titleBar() *Panel {
	title := NewLabel(m.Title)
	title.Style.Bold, title.Style.FontSize, title.Style.FlexGrow = true, FontBody+4, 1
	bar := NewRow(12, title)
	bar.Style.Padding = geom.XY(14, 28)

	// The first window button restores when minimized, else minimizes.
	add := func(icon gfx.IconKind, f func()) {
		b := NewIconButton(icon)
		b.OnClick = func(e *Event) { f() }
		bar.AppendChild(b)
	}
	if m.state == ModalMinimized {
		add(gfx.IconRestore, func() { m.SetState(ModalNormal) })
	} else if m.controls&ModalMinimize != 0 {
		add(gfx.IconMinus, func() { m.SetState(ModalMinimized) })
	}
	if m.controls&ModalMaximize != 0 && m.state != ModalMinimized {
		if m.state == ModalMaximized {
			add(gfx.IconRestore, func() { m.SetState(ModalNormal) })
		} else {
			add(gfx.IconMaximize, func() { m.SetState(ModalMaximized) })
		}
	}
	if m.controls&ModalClose != 0 {
		add(gfx.IconClose, m.Cancel)
	}
	return bar
}

func (m *Modal) footer() *Panel {
	row := NewRow(16)
	row.Style.JustifyContent = JustifyEnd
	row.Style.Padding = geom.XY(18, 28)
	if m.buttons == ModalOKCancel {
		cancel := NewButton(m.CancelText)
		cancel.Style.MinWidth = modalButtonMinW
		cancel.OnClick = func(e *Event) { m.Cancel() }
		row.AppendChild(cancel)
	}
	ok := NewButton(m.OKText)
	ok.SetVariant(VariantPrimary)
	ok.Style.MinWidth = modalButtonMinW
	ok.OnClick = func(e *Event) { m.Confirm() }
	row.AppendChild(ok)
	return row
}

// layoutModal places the open modal on the screen.
func (d *Document) layoutModal() {
	m := d.modal
	if m.state == ModalMaximized {
		layoutWidget(m, geom.Rect{W: d.W, H: d.H})
		return
	}
	w, ok := resolve(m.Style.Width, d.W)
	if !ok {
		w = modalDefaultW
	}
	w = geom.Min(w, d.W-2*modalMargin)
	h := geom.Min(measureForced(m, w, w).H, d.H-2*modalMargin)
	y := (d.H - h) / 2
	if m.state == ModalMinimized {
		y = d.H - h - modalMargin
	}
	layoutWidget(m, geom.R((d.W-w)/2, y, w, h))
}
