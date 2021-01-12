package fileinfo

import (
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/swampapp/swamp/internal/ui/component"
	"github.com/swampapp/swamp/internal/ui/flview"
)

type FileInfo struct {
	*component.Component
	*gtk.Box
}

func New(f flview.File) *FileInfo {
	fi := &FileInfo{Component: component.New("/ui/fileinfo")}
	fi.Box = fi.GladeWidget("container").(*gtk.Box)

	lblSize := fi.GladeWidget("sizeLBL").(*gtk.Label)
	lblSize.SetText(f.HSize)

	lblName := fi.GladeWidget("nameLBL").(*gtk.Label)
	lblName.SetText(f.Name)

	lblID := fi.GladeWidget("idLBL").(*gtk.Label)
	lblID.SetText(f.ID)

	lblPath := fi.GladeWidget("pathLBL").(*gtk.Label)
	lblPath.SetText(f.Path)

	lblBhash := fi.GladeWidget("bhashLBL").(*gtk.Label)
	lblBhash.SetText(f.BHash)
	return fi
}

func NewWindow(f flview.File) *gtk.Window {
	box := New(f)
	w, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	w.SetDefaultSize(700, 250)
	w.Add(box)

	btn := box.GladeWidget("closeBTN").(*gtk.Button)
	btn.Connect("clicked", func() bool {
		w.Destroy()
		return true
	})

	w.Connect("key-press-event", func(w *gtk.Window, ev *gdk.Event) bool {
		kp := gdk.EventKeyNewFromEvent(ev)
		switch kp.KeyVal() {
		case gdk.KEY_Escape:
			w.Destroy()
			return true
		default:
			return false
		}
	})

	return w
}
