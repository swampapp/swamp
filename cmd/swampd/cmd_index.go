package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"net/http"
	_ "net/http/pprof"

	"github.com/arl/statsviz"
	"github.com/briandowns/spinner"
	"github.com/labstack/echo/v4"
	"github.com/muesli/reflow/padding"
	"github.com/muesli/reflow/truncate"
	"github.com/rubiojr/rindex"
	"github.com/swampapp/swamp/internal/settings"
	"github.com/urfave/cli/v2"
)

var tStart = time.Now()

const statusStrLen = 40

var socketPath = filepath.Join(settings.DataDir(), "indexing.sock")
var socketClient *http.Client
var once sync.Once

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

func socketServer(progress chan rindex.IndexStats) error {
	e := echo.New()
	e.HideBanner = true
	e.Logger.SetOutput(io.Discard)

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "swampd daemon socket")
	})

	e.GET("/ping", func(c echo.Context) error {
		return c.String(http.StatusOK, "pong")
	})

	var stats rindex.IndexStats
	go func() {
		for stats = range progress {
			time.Sleep(100 * time.Millisecond)
		}
	}()

	e.GET("/stats", func(c echo.Context) error {
		return c.JSON(http.StatusOK, stats)
	})

	log.Debug().Msgf("swampd socket path: %s", socketPath)
	unixListener, err := net.Listen("unix", socketPath)
	if err != nil {
		panic(err)
	}

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, os.Kill, syscall.SIGTERM)
	go func(c chan os.Signal) {
		sig := <-c
		log.Printf("shutting down socket server", sig)
		unixListener.Close()
		os.Exit(0)
	}(sigc)

	e.Listener = unixListener

	log.Print("unix socket server starting")
	e.Start("")

	return nil
}

func sClient() *http.Client {
	once.Do(func() {
		unixDial := func(proto, addr string) (conn net.Conn, err error) {
			return net.Dial("unix", socketPath)
		}
		tr := &http.Transport{
			Dial: unixDial,
		}
		socketClient = &http.Client{Transport: tr}

	})

	return socketClient
}

func indexRepo(cli *cli.Context) error {
	_, err := os.Stat(socketPath)
	if err != nil {
		if !os.IsNotExist(err) {
			panic(err)
		}
	} else {
		if isIndexing() {
			return errors.New("swampd is already running")
		}
		log.Warn().Msgf("socket file found in %s, but looks stale, removing", socketPath)
		os.Remove(socketPath)
	}

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

	go socketServer(progress)

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

	return os.Remove(socketPath)
}

func progressMonitor(logErrors bool, progress chan rindex.IndexStats) {
	s := spinner.New(spinner.CharSets[11], 200*time.Millisecond)
	s.Color("fgGreen")
	fmt.Println()
	s.Suffix = " Analyzing the repository..."

	for {
		p, err := getStats()
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

func isIndexing() bool {
	resp, err := sClient().Get("http://localhost/ping")
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	return string(b) == "pong"
}

func getStats() (rindex.IndexStats, error) {
	resp, err := sClient().Get("http://localhost/stats")
	if err != nil {
		log.Print("error fetching stats")
	}
	defer resp.Body.Close()

	var p rindex.IndexStats
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return p, err
	}

	err = json.Unmarshal(b, &p)

	return p, err
}
