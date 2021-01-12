package downloadlist

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/rs/zerolog/log"
	"github.com/swampapp/swamp/internal/downloader"
	"github.com/swampapp/swamp/internal/resources"
	"github.com/swampapp/swamp/internal/status"
	"github.com/swampapp/swamp/internal/ui/component"
	"github.com/swampapp/swamp/internal/ui/fileinfo"
	"github.com/swampapp/swamp/internal/ui/flview"
	"github.com/swampapp/swamp/internal/ui/tagger"
)

type DownloadList struct {
	*component.Component
	*gtk.Box
	treeView    *flview.FLView
	searchEntry *gtk.SearchEntry
}

func New() *DownloadList {
	d := &DownloadList{Component: component.New("/ui/downloadlist")}
	d.setup()
	d.Box = d.GladeWidget("downloadlistContainer").(*gtk.Box)
	filelistSW := d.GladeWidget("downloadlistSW").(*gtk.ScrolledWindow)
	filelistSW.Add(d.treeView)

	return d
}

func (d *DownloadList) setup() {
	d.treeView = flview.New()
	d.treeView.Connect("row-activated", d.listRowActivated)
	d.treeView.Connect("realize", d.isShown)
	d.treeView.Connect("key-press-event", func(tree *gtk.TreeView, ev *gdk.Event) bool {
		kp := gdk.EventKeyNewFromEvent(ev)
		cntrlMask := uint(gdk.CONTROL_MASK)
		state := kp.State()
		cntrl := state&cntrlMask == cntrlMask
		switch kp.KeyVal() {
		case gdk.KEY_Delete, gdk.KEY_BackSpace:
			d.removeSelected()
			return true
		case gdk.KEY_t:
			if cntrl {
				d.tagSelected()
			}
			return true
		case gdk.KEY_i:
			if cntrl {
				d.showInfo()
			}
			return true
		case gdk.KEY_o:
			if cntrl {
				d.openSelected()
			}
			return true
		default:
			return false
		}
	})
	d.treeView.SetEnableSearch(false)

	d.searchEntry = d.GladeWidget("searchEntry").(*gtk.SearchEntry)
	d.searchEntry.SetCanDefault(true)
	d.searchEntry.GrabFocus()

	d.searchEntry.Connect("search-changed", func() {
		t, _ := d.searchEntry.GetText()
		d.updateFileList(t)
	})

	d.treeView.Connect("button-press-event", func(tree *gtk.TreeView, ev *gdk.Event) bool {
		btn := gdk.EventButtonNewFromEvent(ev)
		switch btn.Button() {
		case gdk.BUTTON_SECONDARY:
			d.secondButtonPressed(btn)
			return true
		default:
			return false
		}
	})
}

func (d *DownloadList) secondButtonPressed(btn *gdk.EventButton) {
	menu, _ := gtk.MenuNew()

	item := menuItem("Open", "action-open")
	item.Connect("activate", func() bool {
		d.openSelected()
		return true
	})
	menu.Add(item)

	item = menuItem("Tag", "action-tag")
	item.Connect("activate", func() bool {
		d.tagSelected()
		return true
	})
	menu.Add(item)

	item = menuItem("Favorite", "action-favorite")
	item.Connect("activate", func() bool {
		d.addFavorite()
		return true
	})
	menu.Add(item)

	item = menuItem("Delete", "action-delete")
	item.Connect("activate", func() bool {
		d.removeSelected()
		return true
	})
	menu.Add(item)

	menu.ShowAll()
	menu.PopupAtPointer(btn.Event)
	menu.GrabFocus()
}

func (d *DownloadList) addFavorite() {
}

func menuItem(label string, image string) *gtk.MenuItem {
	item, _ := gtk.MenuItemNew()
	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 16)
	box.SetHExpand(true)
	box.Add(resources.ScaledImage(24, 24, image))
	lbl, _ := gtk.LabelNew(label)
	box.Add(lbl)
	item.Add(box)

	return item
}

func (d *DownloadList) updateFileList(query string) {
	log.Print("downloadlist: searching for ", query)
	d.treeView.Clear()

	// FIXME
	docs, err := downloader.Instance().Downloaded()
	if err != nil {
		panic(err)
	}

	// FIXME: re-add
	//mainwindow.Instance().SetStatusRight(fmt.Sprintf("%d downloads", len(docs)))

	for _, doc := range docs {
		match, _ := filepath.Match(fmt.Sprintf("*%s*", strings.ToLower(query)), strings.ToLower(doc.Name))
		if query == "*" || match {
			d.treeView.AddRow(resources.ImageForDoc(doc.Name), doc.Name, doc.Path, doc.Size, doc.ID, doc.BHash)
		}
	}
}

func (f *DownloadList) showInfo() {
	files := f.treeView.SelectedFiles()
	if len(files) == 0 {
		return
	}

	w := fileinfo.NewWindow(files[0])
	w.ShowAll()
}

func (d *DownloadList) tagSelected() {
	var fid string
	files := d.treeView.SelectedFiles()
	for _, file := range files {
		fid = file.ID
		break
	}

	tw, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	tw.Add(tagger.New(fid))
	tw.Connect("key-press-event", func(w *gtk.Window, ev *gdk.Event) bool {
		kp := gdk.EventKeyNewFromEvent(ev)
		switch kp.KeyVal() {
		case gdk.KEY_Escape:
			tw.Destroy()
			return true
		default:
			return false
		}
	})
	tw.ShowAll()
}

func (d *DownloadList) openSelected() {
	files := d.treeView.SelectedFiles()
	for _, file := range files {
		if err := downloader.Open(file.ID); err != nil {
			status.Set(fmt.Sprintf("⚠️ error opening %s", file.Name))
		}
	}
}

func (d *DownloadList) removeSelected() {
	var wg sync.WaitGroup
	var removed []flview.File
	var lock sync.Mutex

	files := d.treeView.SelectedFiles()
	for _, file := range files {
		wg.Add(1)
		go func(f flview.File) {
			err := downloader.Instance().Remove(f.ID)
			if err != nil {
				status.Set(fmt.Sprintf("⚠️ error deleting %s", f.Name))
			} else {
				lock.Lock()
				removed = append(removed, f)
				lock.Unlock()
				status.Set(fmt.Sprintf("%s removed", f.Name))
			}
			wg.Done()
		}(file)
	}

	wg.Wait()
	d.treeView.Remove(removed)
}

func (d *DownloadList) isShown(tree *gtk.TreeView) {
	d.updateFileList("*")
	d.searchEntry.GrabFocus()
}

func (d *DownloadList) listRowActivated(tree *gtk.TreeView, path *gtk.TreePath, col *gtk.TreeViewColumn) {
	model, err := tree.GetModel()
	if err != nil {
		panic(err)
	}

	list := model.(*gtk.ListStore)
	iter, _ := list.GetIter(path)
	value, _ := list.GetValue(iter, 4)
	fid, _ := value.GetString()

	err = downloader.Open(fid)
	if err != nil {
		log.Print("error opening ", fid)
	}
}
