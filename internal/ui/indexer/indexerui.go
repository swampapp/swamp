package indexer

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/rs/zerolog/log"
	"github.com/rubiojr/rindex"
	"github.com/swampapp/swamp/internal/indexer"
	"github.com/swampapp/swamp/internal/resources"
	"github.com/swampapp/swamp/internal/ui/component"
)

type Indexer struct {
	*component.Component
	*gtk.Box
	indexAnimation        *gtk.Image
	indexButton           *gtk.Button
	ctx                   context.Context
	cancelFunc            context.CancelFunc
	statusLbl, cpuTimeLbl *gtk.Label
	durationLbl, rssLbl   *gtk.Label
	startTimeLbl          *gtk.Label
	statusProgress        *gtk.ProgressBar
}

func New() *Indexer {
	i := &Indexer{Component: component.New("/ui/indexer")}
	i.Box = i.GladeWidget("container").(*gtk.Box)
	i.statusLbl = i.GladeWidget("statusLbl").(*gtk.Label)
	i.rssLbl = i.GladeWidget("rssLbl").(*gtk.Label)
	i.cpuTimeLbl = i.GladeWidget("cpuTimeLbl").(*gtk.Label)
	i.durationLbl = i.GladeWidget("durationLbl").(*gtk.Label)
	i.startTimeLbl = i.GladeWidget("startTimeLbl").(*gtk.Label)
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
		i.cancelFunc()
	})

	indexer.Daemon().OnStart(func() {
		i.start()
	})

	return i
}

func (i *Indexer) stop() {
	log.Print("indexerui: stop")
	i.statusProgress.SetFraction(1)
	i.indexButton.SetLabel("Start Indexing")
	resources.UpdateImageFromResource(i.indexAnimation, "indexing-done")
	i.statusProgress.SetText("Indexing finished")
	i.indexAnimation.Show()
}

func (i *Indexer) updateTopLabel(pstats indexer.ProcStats, start time.Time, stats rindex.IndexStats) {
	rss := humanize.Bytes(pstats.RSS)
	cpuTime := strconv.FormatUint(pstats.CpuTime, 10)
	duration := strconv.FormatFloat(time.Since(start).Seconds(), 'f', 0, 64)
	i.rssLbl.SetText("Mem: " + rss)
	i.cpuTimeLbl.SetText(fmt.Sprintf("CPU: %ss", cpuTime))
	i.durationLbl.SetText(fmt.Sprintf("Elapsed: %ss", duration))
	i.startTimeLbl.SetText("Indexing started on " + pstats.StartTime.Local().Format("Jan 2 15:04 2006"))
}

func (i *Indexer) start() {
	log.Print("indexerui: start")
	i.ctx, i.cancelFunc = context.WithCancel(context.Background())
	sTime := time.Now()

	glib.IdleAdd(func() {
		i.indexButton.SetLabel("Stop Indexing")
		i.statusProgress.SetText("Preparing to index repository...")
		resources.UpdateImageFromResource(i.indexAnimation, "indexing")
		i.statusLbl.SetText(fmt.Sprintf("Added Files: %d      Snapshots: [%d/%d]",
			0,
			0,
			0,
		))
	})

	go func() {
		for {
			select {
			case <-i.ctx.Done():
				glib.IdleAdd(func() {
					i.stop()
				})
				return
			default:
				// give swampd some time to start
				time.Sleep(1 * time.Second)

				pstats, err := indexer.GetProcStats()
				if err != nil {
					log.Error().Err(err).Msg("error fetching swampd procstats")
				}
				if pstats.RSS == 0 {
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
					i.statusLbl.SetText(fmt.Sprintf("Added Files: %d      Snapshots [%d/%d]",
						stats.IndexedFiles,
						stats.ScannedSnapshots,
						stats.TotalSnapshots,
					))
					i.statusProgress.SetFraction(percentage)
					i.updateTopLabel(pstats, sTime, stats)
				})
			}
		}
	}()
}
