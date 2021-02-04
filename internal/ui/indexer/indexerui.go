package indexer

import (
	"context"
	"fmt"
	"time"

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
	ctx            context.Context
	cancelFunc     context.CancelFunc
	statusLbl      *gtk.Label
	statusProgress *gtk.ProgressBar
}

func New() *Indexer {
	i := &Indexer{Component: component.New("/ui/indexer")}
	i.Box = i.GladeWidget("container").(*gtk.Box)
	i.statusLbl = i.GladeWidget("statusLbl").(*gtk.Label)
	i.statusProgress = i.GladeWidget("statusProgress").(*gtk.ProgressBar)
	i.ctx, i.cancelFunc = context.WithCancel(context.Background())

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
		i.statusProgress.SetText("Snapshot Progress: 0%")
		resources.UpdateImageFromResource(i.indexAnimation, "indexing")
		i.statusLbl.SetText(fmt.Sprintf("Scanned Files: %d       Indexed Files: %d",
			0,
			0,
		))
		go func() {
			for {
				select {
				case <-i.ctx.Done():
					glib.IdleAdd(func() {
						i.statusProgress.SetFraction(1)
					})
					return
				default:
					stats, err := indexer.Stats()
					if err != nil {
						log.Error().Err(err).Msg("indexerui: error retrieving indexer stats")
					}
					percentage := float64(0)
					if stats.CurrentSnapshotTotalFiles > 0 {
						percentage = float64(stats.CurrentSnapshotFiles) / float64(stats.CurrentSnapshotTotalFiles)
					}
					i.statusProgress.SetText(fmt.Sprintf("Snapshot Progress: %d%%", int(percentage*100)))
					glib.IdleAdd(func() {
						i.statusLbl.SetText(fmt.Sprintf("Scanned Files: %d       Indexed Files: %d      Missing Snapshots: %d",
							stats.ScannedFiles,
							stats.IndexedFiles,
							stats.MissingSnapshots+1,
						))
						i.statusProgress.SetFraction(percentage)
					})
					time.Sleep(1 * time.Second)
				}
			}
		}()
	} else {
		resources.UpdateImageFromResource(i.indexAnimation, "indexing-done")
	}

	indexer.EnableDebugging(true)

	indexer.Daemon().OnStop(func() {
		glib.IdleAdd(func() {
			i.cancelFunc()
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
