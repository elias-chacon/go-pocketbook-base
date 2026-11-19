package widget

import (
	"fmt"

	"go-pocketbook-base/internal/toolkit/gfx"
)

// AppChrome is the standard top bar of an app: one menu, named after the app, with
// About (a modal with the name, the description and the version) and Exit (back to the
// Launcher or the home screen). Every app of the ecosystem has it, as the Hello Go
// showcase does, so people find the same two things in the same place.
type AppChrome struct {
	Title, Description, Version string
	// Labels, already translated. LabelVersion has one %s for the version.
	LabelAbout, LabelExit, LabelVersion string
	// Exit leaves the app (Env.Exit); nil hides the Exit item.
	Exit func()
}

// Enabled reports whether there is anything to show: a chrome with no title is not drawn.
func (c AppChrome) Enabled() bool { return c.Title != "" }

// Bar builds the menu bar for a document. Build one for each screen: a widget has a
// single parent.
func (c AppChrome) Bar(doc *Document) *MenuBar {
	items := []MenuItem{{Label: c.LabelAbout, Icon: gfx.IconSearch, OnSelect: func() { c.about(doc) }}}
	if c.Exit != nil {
		items = append(items, MenuItem{Separator: true}, MenuItem{Label: c.LabelExit, Icon: gfx.IconClose, OnSelect: c.Exit})
	}
	bar := NewMenuBar(&Menu{Label: c.Title, Items: items})
	bar.ID = "chrome-bar"
	return bar
}

// Wrap puts the bar above a screen's root and returns the new root. A chrome that is
// not enabled leaves the root as it is.
func (c AppChrome) Wrap(doc *Document, root Widget) Widget {
	if !c.Enabled() {
		return root
	}
	root.Elem().Style.FlexGrow = 1
	outer := NewColumn(0, c.Bar(doc), root)
	outer.Style.Background = Paper
	return outer
}

func (c AppChrome) about(doc *Document) {
	name := NewLabel(c.Title)
	name.Style.Bold, name.Style.FontSize = true, FontTitle
	lines := []Widget{name}
	if c.Description != "" {
		lines = append(lines, NewLabel(c.Description))
	}
	if c.Version != "" {
		v := NewLabel(fmt.Sprintf(c.LabelVersion, c.Version))
		v.Style.Color = Muted
		lines = append(lines, v)
	}
	m := NewModal(c.LabelAbout, lines...)
	m.ID = "modal-about"
	m.Open(doc)
}
