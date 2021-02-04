package indexer

import (
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/rs/zerolog/log"
	"github.com/swampapp/swamp/internal/indexer"
	"github.com/swampapp/swamp/internal/resources"
	"github.com/swampapp/swamp/internal/ui/component"
)

type Indexer struct {
	*component.Component
	*gtk.Box
	indexAnimation *gtk.Image
	indexButton    *gtk.Button
}

func New() *Indexer {
	i := &Indexer{Component: component.New("/ui/indexer")}
	i.Box = i.GladeWidget("container").(*gtk.Box)

	i.indexButton = i.GladeWidget("indexBTN").(*gtk.Button)
	i.indexButton.Connect("clicked", func() {
		lbl, _ := i.indexButton.GetLabel()
		if lbl == "Start Indexing" {
			log.Print("manual indexing request")
			indexer.Daemon().Start()
		} else {
			log.Print("manual indexing stop")
			indexer.Daemon().Stop()
		}
	})

	i.indexAnimation, _ = i.GladeWidget("indexingFlask").(*gtk.Image)
	if indexer.IsRunning() {
		i.indexButton.SetLabel("Stop Indexing")
		resources.UpdateImageFromResource(i.indexAnimation, "indexing")
	} else {
		resources.UpdateImageFromResource(i.indexAnimation, "indexing-done")
	}

	indexer.EnableDebugging(true)

	indexer.Daemon().OnStop(func() {
		glib.IdleAdd(func() {
			i.indexButton.SetLabel("Start Indexing")
			resources.UpdateImageFromResource(i.indexAnimation, "indexing-done")
			i.indexAnimation.Show()
		})
	})

	indexer.Daemon().OnStart(func() {
		//log.Print("indexerui: notification received")
		glib.IdleAdd(func() {
			i.indexButton.SetLabel("Stop Indexing")
			resources.UpdateImageFromResource(i.indexAnimation, "indexing")
			i.indexAnimation.Show()
		})
	})

	return i
}
