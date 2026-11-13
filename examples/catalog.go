// examples/catalog.go
package examples

import (
	"strings"

	"go-pocketbook-base/internal/gfx"
	"go-pocketbook-base/internal/widget"
)

// Catalog is a searchable, sortable table with a detail card.
type Catalog struct {
	Root   widget.Widget
	Search *widget.Input
	Table  *widget.Table
	Detail *widget.Label

	all [][]string // every row; the table shows the filtered ones
}

// NewCatalog builds the screen; each row is {title, author, pages}.
func NewCatalog(rows [][]string) *Catalog {
	c := &Catalog{
		all:    rows,
		Search: widget.NewSearchInput("Search by title"),
		Detail: widget.NewLabel("Select a book"),
		Table: widget.NewTable(4,
			widget.TableColumn{Title: "Title", Flex: 3, Sortable: true},
			widget.TableColumn{Title: "Author", Flex: 2, Sortable: true},
			widget.TableColumn{Title: "Pages", Flex: 1, Align: gfx.AlignEnd, Sortable: true}),
	}
	c.Search.OnSubmit = func(e *widget.Event) { c.filter(c.Search.Value) }
	c.Search.OnChange = func(e *widget.Event) {
		if c.Search.Value == "" { // the clear (×) button
			c.filter("")
		}
	}
	c.Table.OnSelect = func(e *widget.Event) {
		row := c.Table.Rows[c.Table.Selected]
		c.Detail.Text = row[0] + " — " + row[1] + " (" + row[2] + " pages)"
	}
	c.filter("")

	c.Root = widget.NewColumn(20, c.Search, c.Table, widget.NewCard("Detail", "", c.Detail))
	return c
}

// filter keeps the rows whose title contains query (case-insensitive).
func (c *Catalog) filter(query string) {
	var rows [][]string
	for _, r := range c.all {
		if strings.Contains(strings.ToLower(r[0]), strings.ToLower(query)) {
			rows = append(rows, r)
		}
	}
	c.Table.Rows, c.Table.Page, c.Table.Selected = rows, 0, -1
}
