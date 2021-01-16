package flview

import (
	"fmt"
	"log"
	"strconv"

	"github.com/dustin/go-humanize"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
	"github.com/swampapp/swamp/internal/status"
)

type FLView struct {
	*gtk.TreeView
	listStore *gtk.ListStore
}

type File struct {
	Name  string
	ID    string
	BHash string
	Path  string
	Size  uint64
	HSize string
}

type ColID int

const (
	COLUMN_ICON ColID = iota
	COLUMN_NAME
	COLUMN_PATH
	COLUMN_SIZE
	COLUMN_ID
	COLUMN_USIZE
	COLUMN_BHASH
)

func New() *FLView {
	flv := &FLView{}
	flv.TreeView, _ = gtk.TreeViewNew()
	flv.listStore, _ = gtk.ListStoreNew(
		glib.TYPE_OBJECT,
		glib.TYPE_STRING,
		glib.TYPE_STRING,
		glib.TYPE_STRING,
		glib.TYPE_STRING,
		glib.TYPE_INT,
		glib.TYPE_STRING,
	)
	flv.SetModel(flv.listStore)

	selection, _ := flv.GetSelection()
	selection.SetMode(gtk.SELECTION_MULTIPLE)

	flv.Set("activate-on-single-click", false)

	flv.AppendColumn(createImageColumn("", int(COLUMN_ICON)))
	flv.AppendColumn(createColumn("Filename", int(COLUMN_NAME), 60))
	flv.AppendColumn(createColumn("Path", int(COLUMN_PATH), 40))
	flv.AppendColumn(createBytesColumn("Size", int(COLUMN_SIZE), 40))
	flv.AppendColumn(createColumn("ID", int(COLUMN_ID), 40))
	flv.AppendColumn(createColumn("BHash", int(COLUMN_BHASH), 40))
	flv.SetEnableSearch(false)

	// Creating a list store. This is what holds the data that will be shown on our tree view.

	return flv
}

func (flv *FLView) RemoveSelected() {
	sel, _ := flv.GetSelection()
	rows := sel.GetSelectedRows(flv.Model())
	rows.Foreach(func(item interface{}) {
		path := item.(*gtk.TreePath)
		if iter, err := flv.Model().GetIter(path); err == nil {
			value, _ := flv.Model().GetValue(iter, int(COLUMN_NAME))
			name, _ := value.GetString()
			status.Set(fmt.Sprintf("%s removed", name))
			glib.IdleAdd(func() {
				flv.Model().Remove(iter)
			})
		}
	})
}

func (flv *FLView) Remove(files []File) {
	fmap := map[string]struct{}{}
	for _, f := range files {
		fmap[f.ID] = struct{}{}
	}

	sel, _ := flv.GetSelection()
	rows := sel.GetSelectedRows(flv.Model())
	rows.Foreach(func(item interface{}) {
		path := item.(*gtk.TreePath)
		if iter, err := flv.Model().GetIter(path); err == nil {
			value, _ := flv.Model().GetValue(iter, int(COLUMN_ID))
			id, _ := value.GetString()
			if _, ok := fmap[id]; ok {
				status.Set(fmt.Sprintf("%s removed", id))
				glib.IdleAdd(func() {
					flv.Model().Remove(iter)
				})
			}
		}
	})
}

func (flv *FLView) SelectedFiles() []File {
	files := []File{}

	sel, _ := flv.GetSelection()
	rows := sel.GetSelectedRows(flv.Model())
	rows.Foreach(func(item interface{}) {
		path := item.(*gtk.TreePath)
		if iter, err := flv.Model().GetIter(path); err == nil {
			file := File{}
			value, _ := flv.Model().GetValue(iter, int(COLUMN_ID))
			file.ID, _ = value.GetString()
			value, _ = flv.Model().GetValue(iter, int(COLUMN_NAME))
			file.Name, _ = value.GetString()
			value, _ = flv.Model().GetValue(iter, int(COLUMN_PATH))
			file.Path, _ = value.GetString()
			value, _ = flv.Model().GetValue(iter, int(COLUMN_SIZE))
			file.HSize, _ = value.GetString()
			file.Size, _ = humanize.ParseBytes(file.HSize)
			value, _ = flv.Model().GetValue(iter, int(COLUMN_BHASH))
			file.BHash, _ = value.GetString()
			files = append(files, file)
		}
	})

	return files
}

func (flv *FLView) FileAt(row int) (*File, error) {
	path, _, _, _, _ := flv.GetPathAtPos(0, row)
	iter, err := flv.Model().GetIter(path)
	if err != nil {
		return nil, err
	}
	fid := &File{}

	value, _ := flv.Model().GetValue(iter, int(COLUMN_NAME))
	fid.Name, _ = value.GetString()

	value, _ = flv.Model().GetValue(iter, int(COLUMN_ID))
	fid.ID, _ = value.GetString()

	return fid, nil
}

func (flv *FLView) ItemCount() int {
	return flv.Model().IterNChildren(nil)
}

func (flv *FLView) Clear() {
	flv.Model().Clear()
}

func (flv *FLView) Model() *gtk.ListStore {
	return flv.listStore
}

func (flv *FLView) TreeViewWidget() *gtk.TreeView {
	return flv.TreeView
}

func (flv *FLView) AddRow(image *gdk.Pixbuf, filename, path, size, fileID, bhash string) {
	iter := flv.Model().Append()

	// Set the contents of the list store row that the iterator represents
	usize, _ := strconv.ParseUint(size, 10, 64)
	// the 5 column is an invisible column used to store the size in bytes, so it can be
	// properly sorted when clicking the column
	err := flv.Model().Set(iter,
		[]int{int(COLUMN_ICON), int(COLUMN_NAME), int(COLUMN_PATH), int(COLUMN_SIZE), int(COLUMN_ID), int(COLUMN_USIZE), int(COLUMN_BHASH)},
		[]interface{}{image, filename, path, humanize.Bytes(usize), fileID, usize, bhash})

	if err != nil {
		log.Print("Unable to add row")
		panic(err)
	}
}

// Add a column to the tree view (during the initialization of the tree view)
func createColumn(title string, id int, width int) *gtk.TreeViewColumn {
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

// Add a column to the tree view (during the initialization of the tree view)
func createBytesColumn(title string, id int, width int) *gtk.TreeViewColumn {
	cellRenderer, _ := gtk.CellRendererTextNew()
	cellRenderer.Set("xpad", 20)
	cellRenderer.Set("ellipsize-set", true)
	cellRenderer.Set("ellipsize", pango.ELLIPSIZE_END)
	cellRenderer.Set("width-chars", width)

	column, _ := gtk.TreeViewColumnNewWithAttribute(title, cellRenderer, "text", id)
	column.SetResizable(true)
	column.SetSortColumnID(5)
	column.SetSortOrder(gtk.SORT_DESCENDING)

	return column
}

func createImageColumn(title string, id int) *gtk.TreeViewColumn {
	// In this column we want to show image data from Pixbuf, hence
	// create a pixbuf renderer
	cellRenderer, _ := gtk.CellRendererPixbufNew()

	// Tell the renderer where to pick input from. Pixbuf renderer understands
	// the "pixbuf" property.
	column, _ := gtk.TreeViewColumnNewWithAttribute(title, cellRenderer, "pixbuf", id)

	return column
}
