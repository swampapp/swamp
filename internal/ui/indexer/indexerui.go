package indexer

import (
	"context"
	"fmt"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/prometheus/procfs"
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

	indexer.EnableDebugging(true)

	indexer.Daemon().OnStop(func() {
		glib.IdleAdd(func() {
			i.cancelFunc()
		})
	})

	indexer.Daemon().OnStart(func() {
		i.start()
	})

	return i
}

func (i *Indexer) stop() {
	i.statusProgress.SetFraction(1)
	i.indexButton.SetLabel("Start Indexing")
	resources.UpdateImageFromResource(i.indexAnimation, "indexing-done")
	i.statusProgress.SetText("Indexing finished")
	i.indexAnimation.Show()
}

func (i *Indexer) swampdRSS() string {
	pid, err := indexer.Pid()
	pstats := procfs.ProcStat{}
	if err != nil {
		log.Error().Err(err).Msg("error getting swampd PID")
		return "0"
	}

	p, err := procfs.NewProc(pid)
	if err != nil {
		log.Error().Err(err).Msgf("could not get process: %s", err)
		return "0"
	} else {
		pstats, err = p.NewStat()
		if err != nil {
			log.Error().Err(err).Msgf("could not get process stat: %s", err)
		}
	}

	return humanize.Bytes(uint64(pstats.ResidentMemory()))
}
func (i *Indexer) start() {
	i.indexButton.SetLabel("Stop Indexing")
	i.statusProgress.SetText("Preparing to index repository...")
	resources.UpdateImageFromResource(i.indexAnimation, "indexing")

	rss := i.swampdRSS()
	i.statusLbl.SetText(fmt.Sprintf("swampd memory: %s       New Files: %d      Missing Snapshots: %d",
		rss,
		0,
		0,
	))
	go func() {
		for {
			select {
			case <-i.ctx.Done():
				glib.IdleAdd(func() {
					i.stop()
				})
				return
			default:
				rss = i.swampdRSS()
				if rss == "0" {
					continue
				}
				stats, err := indexer.Stats()
				if err != nil {
					log.Error().Err(err).Msg("indexerui: error retrieving indexer stats")
				}
				percentage := float64(0)
				if stats.CurrentSnapshotTotalFiles > 0 {
					percentage = float64(stats.CurrentSnapshotFiles) / float64(stats.CurrentSnapshotTotalFiles)
				}
				glib.IdleAdd(func() {
					if stats.ScannedFiles > 0 {
						i.statusProgress.SetText(fmt.Sprintf("Snapshot Progress: %d%%", int(percentage*100)))
					}
					i.statusLbl.SetText(fmt.Sprintf("swampd memory: %s       New Files: %d      Snapshots [%d/%d]",
						rss,
						stats.IndexedFiles,
						stats.ScannedSnapshots,
						stats.TotalSnapshots,
					))
					i.statusProgress.SetFraction(percentage)
				})
				time.Sleep(1 * time.Second)
			}
		}
	}()
}
