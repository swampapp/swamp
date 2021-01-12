package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"net/http"
	_ "net/http/pprof"

	"github.com/arl/statsviz"
	"github.com/briandowns/spinner"
	"github.com/muesli/reflow/padding"
	"github.com/muesli/reflow/truncate"
	"github.com/rubiojr/rindex"
	"github.com/urfave/cli/v2"
)

var tStart = time.Now()

const statusStrLen = 40

// higher than this could cause trouble with mem usage and file descriptors
const batchSize = 300

func init() {
	cmd := &cli.Command{
		Name:   "index",
		Usage:  "Index the repository",
		Action: indexRepo,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:     "log-errors",
				Usage:    "Log errors",
				Required: false,
			},
			&cli.BoolFlag{
				Name:     "reindex",
				Usage:    "Re-index snapshots",
				Required: false,
			},
			&cli.BoolFlag{
				Name:     "monitor",
				Usage:    "Monitor progress",
				Required: false,
			},
		},
	}
	appCommands = append(appCommands, cmd)
}

func indexRepo(cli *cli.Context) error {
	statsviz.RegisterDefault()
	go func() {
		log.Print(http.ListenAndServe("localhost:6060", nil))
	}()

	progress := make(chan rindex.IndexStats, 10)
	idx, err := rindex.New(indexPath, globalOptions.Repo, globalOptions.Password)
	if err != nil {
		return err
	}
	idxOpts := rindex.DefaultIndexOptions
	idxOpts.BatchSize = batchSize
	idxOpts.DocumentBuilder = FileDocumentBuilder{}
	idxOpts.Reindex = cli.Bool("reindex")

	if cli.Bool("monitor") {
		go progressMonitor(cli.Bool("log-errors"), progress)
	}

	stats, err := idx.Index(context.Background(), idxOpts, progress)
	if err != nil {
		panic(err)
	}
	fmt.Printf(
		"\n💥 %d indexed, %d already present. %d new snapshots. Took %d seconds.\n",
		stats.IndexedFiles,
		stats.AlreadyIndexed,
		stats.ScannedSnapshots,
		int(time.Since(tStart).Seconds()),
	)
	return nil
}

func progressMonitor(logErrors bool, progress chan rindex.IndexStats) {
	s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
	s.Color("fgGreen")
	fmt.Println()
	s.Suffix = " Analyzing the repository..."
	lastError := ""
	for {
		select {
		case p := <-progress:
			if logErrors {
				if len(p.Errors) > 0 {
					e := p.Errors[len(p.Errors)-1].Error()
					if e != lastError {
						fmt.Println("\n", e)
						lastError = e
					}
				}
			}
			lm := p.LastMatch
			if lm == "" {
				lm = "Searching for files..."
			}
			ls := truncate.StringWithTail(lm, statusStrLen, "...")
			rate := float64(p.ScannedNodes*1000000000) / float64(time.Since(tStart))
			percentage := uint64(0)
			if p.CurrentSnapshotTotalFiles > 0 {
				percentage = (p.CurrentSnapshotFiles * 100) / p.CurrentSnapshotTotalFiles
			}
			s.Suffix = fmt.Sprintf(
				"\033[F🎯  %s\n%d new, %d alredy indexed, %d errors, %.0f f/s, %d scanned (%d%%), %d/%d snapshots",
				padding.String(ls, statusStrLen),
				p.IndexedFiles,
				p.AlreadyIndexed,
				len(p.Errors),
				rate,
				p.CurrentSnapshotFiles,
				percentage,
				p.ScannedSnapshots,
				p.MissingSnapshots,
			)
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}
