package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"net/http"
	_ "net/http/pprof"

	"github.com/arl/statsviz"
	"github.com/briandowns/spinner"
	"github.com/gofiber/fiber/v2"
	"github.com/muesli/reflow/padding"
	"github.com/muesli/reflow/truncate"
	"github.com/prometheus/procfs"
	"github.com/rubiojr/rindex"
	"github.com/swampapp/swamp/internal/indexer"
	"github.com/urfave/cli/v2"
)

var tStart = time.Now()

const statusStrLen = 40

// higher than this could cause trouble with mem usage and file descriptors
const batchSize = 300

var pid int

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

func socketServer(cancel context.CancelFunc, progress chan rindex.IndexStats) error {
	f := fiber.New(
		fiber.Config{
			DisableStartupMessage: true,
		},
	)

	f.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("")
	})

	f.Get("/ping", func(c *fiber.Ctx) error {
		return c.SendString("pong")
	})

	f.Get("/procstats", func(c *fiber.Ctx) error {
		p, err := procfs.NewProc(pid)
		if err != nil {
			log.Error().Err(err).Msgf("could not get process: %s", err)
			return err
		}
		pstats, err := p.Stat()
		if err != nil {
			log.Error().Err(err).Msgf("could not get process stat: %s", err)
			return err
		}

		rss := uint64(pstats.ResidentMemory())

		s := indexer.ProcStats{
			RSS:       rss,
			Duration:  uint64(time.Since(tStart).Seconds()),
			StartTime: tStart,
			CpuTime:   uint64(pstats.CPUTime()),
		}

		return c.JSON(s)
	})

	var stats rindex.IndexStats
	go func() {
		for stats = range progress {
			time.Sleep(100 * time.Millisecond)
		}
	}()

	f.Get("/stats", func(c *fiber.Ctx) error {
		return c.JSON(stats)
	})

	f.Get("/pid", func(c *fiber.Ctx) error {
		return c.SendString(fmt.Sprintf("%d", pid))
	})

	f.Post("/kill", func(c *fiber.Ctx) error {
		log.Debug().Msg("swampd was told to quit")
		cancel()
		return c.SendString("shutting down")
	})

	log.Debug().Msgf("swampd socket path: %s", indexer.SocketPath())
	unixListener, err := net.Listen("unix", indexer.SocketPath())
	if err != nil {
		panic(err)
	}

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, syscall.SIGTERM)
	go func(c chan os.Signal) {
		sig := <-c
		log.Printf("shutting down socket server: %v", sig)
		unixListener.Close()
		os.Exit(0)
	}(sigc)

	err = f.Listener(unixListener)
	if err != nil {
		log.Fatal().Err(err).Msg("error setting custom UNIX listener")
	}

	log.Print("unix socket server starting")
	return f.Listen("")
}

func indexRepo(cli *cli.Context) error {
	indexer.EnableDebugging(cli.Bool("debug"))

	pid = os.Getpid()
	_, err := os.Stat(indexer.SocketPath())
	if err != nil {
		if !os.IsNotExist(err) {
			panic(err)
		}
	} else {
		if indexer.IsRunning() {
			return errors.New("swampd is already running")
		}
		log.Warn().Msgf("socket file found in %s, but looks stale, removing", indexer.SocketPath())
		os.Remove(indexer.SocketPath())
	}

	if err := statsviz.RegisterDefault(); err == nil {
		go func() {
			log.Print(http.ListenAndServe("localhost:6060", nil))
		}()
	} else {
		log.Error().Err(err).Msg("error running statsviz")
	}

	progress := make(chan rindex.IndexStats, 10)
	idx, err := rindex.New(indexPath, globalOptions.Repo, globalOptions.Password)
	if err != nil {
		return err
	}
	idxOpts := rindex.DefaultIndexOptions
	idxOpts.BatchSize = batchSize
	idxOpts.DocumentBuilder = FileDocumentBuilder{}
	idxOpts.Reindex = cli.Bool("reindex")

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		if err := socketServer(cancel, progress); err != nil {
			log.Error().Err(err).Msg("socket server returned an error")
		}
	}()

	if cli.Bool("monitor") {
		go progressMonitor(cli.Bool("log-errors"), progress)
	}

	stats, err := idx.Index(ctx, idxOpts, progress)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			log.Fatal().Err(err).Msg("indexing process aborted with an unknown error")
		}
		log.Print("indexing stopped")
	}

	if cli.Bool("monitor") {
		fmt.Printf(
			"\n💥 %d indexed, %d already present. %d new snapshots. Took %d seconds.\n",
			stats.IndexedFiles,
			stats.AlreadyIndexed,
			stats.ScannedSnapshots,
			int(time.Since(tStart).Seconds()),
		)
	} else {
		log.Info().Msgf(
			"%d indexed, %d already present. %d new snapshots. Took %d seconds.\n",
			stats.IndexedFiles,
			stats.AlreadyIndexed,
			stats.ScannedSnapshots,
			int(time.Since(tStart).Seconds()),
		)

	}
	return os.Remove(indexer.SocketPath())
}

func progressMonitor(logErrors bool, progress chan rindex.IndexStats) {
	s := spinner.New(spinner.CharSets[11], 200*time.Millisecond)
	//nolint
	s.Color("fgGreen")
	fmt.Println()
	s.Suffix = " Analyzing the repository..."

	for {
		p, err := indexer.Stats()
		if err != nil {
			log.Error().Err(err).Msgf("error reading stats")
			continue
		}
		printStats(logErrors, p, s)
		time.Sleep(200 * time.Millisecond)
	}
}

var lastError = ""

func printStats(logErrors bool, p rindex.IndexStats, s *spinner.Spinner) {
	if logErrors {
		if len(p.Errors) > 0 {
			e := p.Errors[len(p.Errors)-1].Error()
			if e != lastError {
				fmt.Println("\n", e)
				lastError = e
			}
		}
	}

	// Wait until a file has been scanned to start printing progress
	if p.ScannedFiles == 0 {
		return
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
}
