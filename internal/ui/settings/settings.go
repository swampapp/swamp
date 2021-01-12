package settings

import (
	"github.com/gotk3/gotk3/gtk"
	"github.com/swampapp/swamp/internal/ui/component"
)

type Settings struct {
	*gtk.Box
	*component.Component
}

func New() *Settings {
	s := &Settings{Component: component.New("/ui/settings")}
	s.Box = s.GladeWidget("container").(*gtk.Box)

	btn := s.GladeWidget("testBTN").(*gtk.Button)
	btn.Connect("clicked", func() bool {
		img, _ := gtk.ImageNewFromResource("/ui/behappy")
		w, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
		w.SetDefaultSize(700, 250)
		w.Add(img)
		w.ShowAll()
		return true
	})

	return s
}
