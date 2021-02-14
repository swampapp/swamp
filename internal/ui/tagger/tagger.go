package tagger

import (
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/swampapp/swamp/internal/logger"
	"github.com/swampapp/swamp/internal/tags"
	"github.com/swampapp/swamp/internal/ui/component"
	"github.com/swampapp/swamp/internal/ui/util"
)

type Tagger struct {
	*component.Component
	*gtk.Box
	listStore *gtk.ListStore
	treeView  *gtk.TreeView
	fileID    string
}

func New(fileID string) *Tagger {
	t := &Tagger{
		Component: component.New("/ui/tagger"),
		fileID:    fileID,
	}
	t.Box = t.GladeWidget("component").(*gtk.Box)

	t.Connect("destroy", func() {
		t.saveTags()
	})

	treeView, _ := gtk.TreeViewNew()
	treeView.Set("activate-on-single-click", false)
	treeView.AppendColumn(util.CreateColumn("Name", 0, 10))
	treeView.SetEnableSearch(true)
	treeView.Connect("key-press-event", func(tree *gtk.TreeView, ev *gdk.Event) bool {
		kp := gdk.EventKeyNewFromEvent(ev)
		switch kp.KeyVal() {
		case gdk.KEY_Delete, gdk.KEY_BackSpace:
			t.removeSelected()
			return true
		default:
			return false
		}
	})

	selection, _ := treeView.GetSelection()
	selection.SetMode(gtk.SELECTION_MULTIPLE)

	listStore, _ := gtk.ListStoreNew(glib.TYPE_STRING)
	treeView.SetModel(listStore)

	t.treeView = treeView
	t.listStore = listStore

	entry := t.GladeWidget("entry").(*gtk.Entry)
	entry.Connect("activate", func(entry *gtk.Entry) bool {
		text, _ := entry.GetText()
		t.AddTag(text)
		return true
	})

	btn := t.GladeWidget("addBtn").(*gtk.Button)
	btn.Connect("clicked", func() bool {
		text, _ := entry.GetText()
		t.AddTag(text)
		return true
	})

	btn = t.GladeWidget("closeBtn").(*gtk.Button)
	btn.Connect("clicked", func() bool {
		t.saveTags()
		t.Destroy()
		return true
	})

	sw := t.GladeWidget("sw").(*gtk.ScrolledWindow)
	sw.Add(treeView)
	sw.SetSizeRequest(100, 200)

	t.populate()

	return t
}

func (t *Tagger) removeSelected() {
	sel, _ := t.treeView.GetSelection()
	rows := sel.GetSelectedRows(t.listStore)
	rows.Foreach(func(item interface{}) {
		path := item.(*gtk.TreePath)
		if iter, err := t.listStore.GetIter(path); err == nil {
			t.listStore.Remove(iter)
		}
	})
}

func (t *Tagger) saveTags() {
	var tl []tags.Tag
	t.listStore.ForEach(func(model *gtk.TreeModel, path *gtk.TreePath, iter *gtk.TreeIter, userdata ...interface{}) bool {
		value, _ := t.listStore.GetValue(iter, 0)
		tname, _ := value.GetString()
		tag := tags.Tag{Name: tname}
		tl = append(tl, tag)
		return false
	})
	err := tags.Save(t.fileID, tl)
	if err != nil {
		logger.Errorf(err, "error saving tags for %s", t.fileID)
	} else {
		logger.Infof("saved tags for %s", t.fileID)
	}
}

func (t *Tagger) populate() {
	t.listStore.Clear()

	tl, err := tags.For(t.fileID)
	if err != nil {
		logger.Errorf(err, "error loading tags for %s", t.fileID)
		return
	}
	for _, tag := range tl {
		logger.Infof("populating with tag %s", tag.Name)
		iter := t.listStore.Append()
		t.listStore.Set(iter,
			[]int{0},
			[]interface{}{tag.Name})
	}
}

// Append a row to the list store for the tree view
func (t *Tagger) AddTag(tag string) {
	iter := t.listStore.Append()

	// Set the contents of the list store row that the iterator represents
	err := t.listStore.Set(iter,
		[]int{0},
		[]interface{}{tag})
	if err != nil {
		panic(err)
	}
}

func (t *Tagger) Destroy() {
	p, _ := t.GetParent()
	(p.(*gtk.Window)).Destroy()
}
