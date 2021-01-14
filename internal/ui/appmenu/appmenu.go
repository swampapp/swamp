package appmenu

import (
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/rs/zerolog/log"
	"github.com/swampapp/swamp/internal/downloader"
	"github.com/swampapp/swamp/internal/resources"
	"github.com/swampapp/swamp/internal/ui/component"
	"github.com/swampapp/swamp/internal/ui/reposelector"
	"github.com/swampapp/swamp/internal/ui/util"
)

type AppMenu struct {
	*component.Component
	*gtk.Box
	treeView         *gtk.TreeView
	listStore        *gtk.ListStore
	selectionHandler func(string)
}

func (a *AppMenu) Widget() gtk.IWidget {
	return a.Box
}

func New() *AppMenu {
	a := &AppMenu{Component: component.New("/ui/appmenu")}
	a.Box = a.GladeWidget("appmenuContainer").(*gtk.Box)
	a.setup()

	sw := a.GladeWidget("appMenuSW").(*gtk.ScrolledWindow)
	sw.Add(a.treeView)
	downloader.Instance().AddObserver(a)

	a.Box.Add(reposelector.New())

	return a
}

func (a *AppMenu) OnSelectionChanged(fn func(string)) {
	a.selectionHandler = fn
}

// Handler of "changed" signal of TreeView's selection
func (a *AppMenu) selectionChanged(s *gtk.TreeSelection) {
	// Returns glib.List of gtk.TreePath pointers
	rows := s.GetSelectedRows(a.listStore)
	//items := make([]string, 0, rows.Length())

	for l := rows; l != nil; l = l.Next() {
		path := l.Data().(*gtk.TreePath)
		iter, _ := a.listStore.GetIter(path)
		value, _ := a.listStore.GetValue(iter, 1)
		str, _ := value.GetString()
		a.selectionHandler(str)
	}
}

// Add a column to the tree view (during the initialization of the tree view)
func (a *AppMenu) createNumberColumn(title string, id int) *gtk.TreeViewColumn {
	cellRenderer, _ := gtk.CellRendererTextNew()
	cellRenderer.Set("font", "Sans Bold 12")

	column, _ := gtk.TreeViewColumnNewWithAttribute(title, cellRenderer, "text", id)

	return column
}

func (a *AppMenu) addRowWithImage(img *gdk.Pixbuf, text string) {
	// Get an iterator for a new row at the end of the list store
	iter := a.listStore.Append()

	// Set the contents of the list store row that the iterator represents
	err := a.listStore.Set(iter,
		[]int{0, 1, 2},
		[]interface{}{img, text, ""})

	if err != nil {
		log.Fatal().Err(err).Msg("unable to add row")
	}
}

func (a *AppMenu) setup() {
	a.treeView, _ = gtk.TreeViewNew()
	a.treeView.SetCanFocus(false)
	a.treeView.Set("headers-visible", false)
	a.treeView.SetEnableSearch(false)

	a.treeView.AppendColumn(util.CreateImageColumn("", 0))
	a.treeView.AppendColumn(util.CreateColumn("Name", 1, 40))
	a.treeView.AppendColumn(a.createNumberColumn("Items", 2))

	// Creating a list store. This is what holds the data that will be shown on our tree view.
	a.listStore, _ = gtk.ListStoreNew(glib.TYPE_OBJECT, glib.TYPE_STRING, glib.TYPE_STRING)
	a.treeView.SetModel(a.listStore)

	selection, err := a.treeView.GetSelection()
	if err != nil {
		panic(err)
	}
	selection.SetMode(gtk.SELECTION_SINGLE)
	selection.Connect("changed", a.selectionChanged)

	scaleFactor := a.treeView.GetScaleFactor()
	var imageTags, imageSearch, imageSettings, imageStatus, imageDownloaded, imageInProgress *gdk.Pixbuf

	if scaleFactor == 1 {
		imageTags = resources.Pixbuf("ui/appmenu/tags.svg")
		imageSettings = resources.Pixbuf("ui/appmenu/settings.svg")
		imageStatus = resources.Pixbuf("ui/appmenu/index.svg")
		imageSearch = resources.Pixbuf("ui/appmenu/search.svg")
		imageDownloaded = resources.Pixbuf("ui/appmenu/downloads.svg")
		imageInProgress = resources.Pixbuf("ui/appmenu/in-progress.svg")
	} else {
		// HiDPI hack while the required cairo stuff is missing in gotk3
		// See https://gitlab.gnome.org/GNOME/gtk/-/issues/613
		imageTags = resources.ScaledPixbuf(32, 32, "ui/appmenu/tags.svg")
		imageSettings = resources.ScaledPixbuf(32, 32, "ui/appmenu/settings.svg")
		imageStatus = resources.ScaledPixbuf(32, 32, "ui/appmenu/index.svg")
		imageSearch = resources.ScaledPixbuf(32, 32, "ui/appmenu/search.svg")
		imageDownloaded = resources.ScaledPixbuf(32, 32, "ui/appmenu/downloads.svg")
		imageInProgress = resources.ScaledPixbuf(32, 32, "ui/appmenu/in-progress.svg")
	}

	// Add some rows to the list store
	a.addRowWithImage(imageSearch, "Search")
	a.addRowWithImage(imageTags, "Tags")
	a.addRowWithImage(imageDownloaded, "Downloaded")
	a.addRowWithImage(imageInProgress, "In Progress")
	a.addRowWithImage(imageStatus, "Indexer")
	a.addRowWithImage(imageSettings, "Settings")
}

// Creates a tree view and the list store that holds its data
func (a *AppMenu) SelectPath(p string) {
	selection, err := a.treeView.GetSelection()
	if err != nil {
		panic(err)
	}
	path, _ := gtk.TreePathNewFromString(p)
	selection.SelectPath(path)
}

// Implements interface to listen for downloader events
func (a *AppMenu) Name() string {
	return "App Menu observer"
}

// Implements interface to listen for downloader events
func (a *AppMenu) NotifyCallback(evt downloader.DownloadEvent) {
	l := downloader.Instance().InProgress()
	glib.IdleAdd(func() {
		iter, _ := a.listStore.GetIterFromString("3:3")
		if l == 0 {
			a.listStore.SetValue(iter, 2, "")
		} else {
			a.listStore.SetValue(iter, 2, l)
		}
	})
}
