package util

import (
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

// Add a column to the tree view (during the initialization of the tree view)
func CreateColumn(title string, id int, width int) *gtk.TreeViewColumn {
	cellRenderer, _ := gtk.CellRendererTextNew()
	cellRenderer.Set("xpad", 20)
	cellRenderer.Set("ellipsize-set", true)
	cellRenderer.Set("ellipsize", pango.ELLIPSIZE_END)
	cellRenderer.Set("width-chars", width)

	column, _ := gtk.TreeViewColumnNewWithAttribute(title, cellRenderer, "text", id)
	column.SetResizable(true)
	column.SetSortColumnID(id)

	return column
}

func CreateImageColumn(title string, id int) *gtk.TreeViewColumn {
	// In this column we want to show image data from Pixbuf, hence
	// create a pixbuf renderer
	cellRenderer, _ := gtk.CellRendererPixbufNew()

	// Tell the renderer where to pick input from. Pixbuf renderer understands
	// the "pixbuf" property.
	column, _ := gtk.TreeViewColumnNewWithAttribute(title, cellRenderer, "pixbuf", id)

	return column
}
