package taglist

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/rs/zerolog/log"
	"github.com/swampapp/swamp/internal/resources"
	"github.com/swampapp/swamp/internal/tags"
	"github.com/swampapp/swamp/internal/ui/component"
	"github.com/swampapp/swamp/internal/ui/util"
)

type TagList struct {
	*component.Component
	*gtk.Box
	treeView       *gtk.TreeView
	listStore      *gtk.ListStore
	tagActionImage *gdk.Pixbuf
	searchEntry    *gtk.SearchEntry
}

type TagSelectedObserver interface {
	TagSelected(string)
}

type TagSelectedCallback = func(tag string)

var observers sync.Map

type ColID int

const (
	COLUMN_ICON ColID = iota
	COLUMN_NAME
	COLUMN_COLOR
)

// Creates a tree view and the list store that holds its data
func New() *TagList {
	t := &TagList{Component: component.New("/ui/taglist")}
	t.tagActionImage = resources.Pixbuf("action-tag")
	t.Box = t.GladeWidget("container").(*gtk.Box)
	filelistSW := t.GladeWidget("taglistSW").(*gtk.ScrolledWindow)
	t.setup()
	filelistSW.Add(t.treeView)

	return t
}

func (t *TagList) setup() {
	t.treeView, _ = gtk.TreeViewNew()

	selection, err := t.treeView.GetSelection()
	if err != nil {
		panic(err)
	}
	selection.SetMode(gtk.SELECTION_SINGLE)

	t.treeView.Set("activate-on-single-click", false)

	t.treeView.AppendColumn(util.CreateImageColumn("", int(COLUMN_ICON)))
	t.treeView.AppendColumn(util.CreateColumn("Name", int(COLUMN_NAME), 80))
	t.treeView.AppendColumn(util.CreateColumn("Color", int(COLUMN_COLOR), 40))
	t.treeView.SetEnableSearch(false)

	// Creating a list store. This is what holds the data that will be shown on our tree view.
	t.listStore, _ = gtk.ListStoreNew(glib.TYPE_OBJECT, glib.TYPE_STRING, glib.TYPE_STRING)
	t.treeView.SetModel(t.listStore)
	t.treeView.Connect("row-activated", t.rowActivated)
	t.treeView.Connect("realize", t.isShown)

	t.treeView.SetEnableSearch(false)

	t.searchEntry = t.GladeWidget("searchEntry").(*gtk.SearchEntry)
	t.searchEntry.SetCanDefault(true)

	t.searchEntry.Connect("search-changed", func() {
		txt, err := t.searchEntry.GetText()
		if err != nil {
			panic(err)
		}
		t.updateFileList(txt)
	})

	t.treeView.Connect("button-press-event", func(tree *gtk.TreeView, ev *gdk.Event) bool {
		btn := gdk.EventButtonNewFromEvent(ev)
		switch btn.Button() {
		case gdk.BUTTON_SECONDARY:
			secondButtonPressed(t.treeView, btn)
			return true
		default:
			return false
		}
	})
}

func secondButtonPressed(treeview *gtk.TreeView, btn *gdk.EventButton) {
}

func (t *TagList) updateFileList(query string) {
	log.Print("taglist: searching for ", query)
	t.listStore.Clear()

	tags, err := tags.All()
	if err != nil {
		log.Error().Err(err).Msg("error listing tags")
		return
	}

	for _, tag := range tags {
		match, _ := filepath.Match(fmt.Sprintf("*%s*", strings.ToLower(query)), strings.ToLower(tag.Name))
		if query == "*" || match {
			t.addTagRow(tag.Name, tag.Color)
		}
	}
}

func (t *TagList) addTagRow(name, color string) {
	// Get an iterator for a new row at the end of the list store
	iter := t.listStore.Append()

	// properly sorted when clicking the column
	err := t.listStore.Set(iter,
		[]int{int(COLUMN_ICON), int(COLUMN_NAME), int(COLUMN_COLOR)},
		[]interface{}{t.tagActionImage, name, color})
	if err != nil {
		log.Print("Unable to add row")
		panic(err)
	}
}

func TagSelectedEvent(l string, cb TagSelectedCallback) {
	observers.Store(l, cb)
}

func TagSelected(tag string) {
	observers.Range(func(key, value interface{}) bool {
		if key == nil {
			return false
		}

		cb := value.(TagSelectedCallback)
		cb(tag)
		return true
	})
}

func (t *TagList) isShown(tree *gtk.TreeView) {
	t.updateFileList("*")
	t.searchEntry.GrabFocus()
}

func (t *TagList) rowActivated(tree *gtk.TreeView, path *gtk.TreePath, col *gtk.TreeViewColumn) {
	iter, _ := t.listStore.GetIter(path)
	value, _ := t.listStore.GetValue(iter, int(COLUMN_NAME))
	tag, _ := value.GetString()

	TagSelected(tag)
}
