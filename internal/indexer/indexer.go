package indexer

// FIXME:
// * Needs to handle graceful shutdowns (i.e. when exiting the app)
// * Use Cancel context to be able to shutdown the indexer gracefully

import (
	"context"
	"os/exec"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/swampapp/swamp/internal/config"
	"github.com/swampapp/swamp/internal/index"
	"github.com/swampapp/swamp/internal/resticsettings"
	"github.com/swampapp/swamp/internal/settings"
	"golang.org/x/sync/semaphore"
)

var sem = semaphore.NewWeighted(1)

type Indexer struct {
}

var once sync.Once
var instance *Indexer
var cancel context.CancelFunc
var ctx context.Context

func init() {
	daemonize()
}

func daemonize() {
	go func() {
		ticker := time.NewTicker(30 * time.Minute)
		for {
			select {
			case <-ticker.C:
				log.Print("indexer: trying to start swampd")
				Daemon().Start()
			}
		}
	}()
}

func Daemon() *Indexer {
	once.Do(func() {
		instance = &Indexer{}
	})

	return instance
}

func (i *Indexer) Start() {
	go func() {
		if !sem.TryAcquire(1) {
			log.Print("indexer: already running, skiping")
			return
		}
		defer sem.Release(1)
		ctx, cancel = context.WithCancel(context.Background())

		prepo := config.PreferredRepo()
		log.Printf("indexer: STARTING the indexing goroutine, repo %s", prepo)
		rs := resticsettings.New(prepo)

		log.Print("indexer: STARTED the indexing goroutine")
		notifyStart()

		for {
			time.Sleep(10 * time.Second)
			// FIXME
			if !resticsettings.FirstBoot() {
				log.Print("indexer: no first boot")
				break
			}
			log.Print("indexer: waiting for first boot")
		}

		rs.ExportEnv()
		defer resticsettings.ResetEnv()

		log.Debug().Msg("indexer: checking for new snapshots")
		ok, err := index.NeedsIndexing(config.PreferredRepo())
		if err != nil {
			log.Error().Err(err).Msg("indexer: error accessing the repository")
		}

		defer func() {
			notifyStop()
			log.Print("indexer: stopped swampd")
		}()

		if ok {
			log.Print("indexer: starting swampd")
			bin, err := exec.LookPath("swampd")
			if err != nil {
				log.Error().Err(err).Msg("error finding swampd executable path")
				return
			}
			log.Printf("indexer: %s %s %s %s", bin, "--index-path", settings.IndexPath(), "index")
			cmd := exec.CommandContext(ctx, bin, "--index-path", settings.IndexPath(), "index")
			out, err := cmd.CombinedOutput()
			if err != nil {
				log.Error().Err(err).Msgf("indexer: swampd error: %s", out)
			}
		} else {
			log.Debug().Msg("indexer: does not need indexing")
		}
	}()
}

func (i *Indexer) Stop() {
	if !sem.TryAcquire(1) {
		log.Print("indexer: trying to stop")
		cancel()
		return
	}
	sem.Release(1)
}

func (i *Indexer) IsRunning() bool {
	if !sem.TryAcquire(1) {
		return true
	}
	sem.Release(1)
	return false
}

type OnStartCb func()

var onStartListeners []OnStartCb

// FIXME: not thread safe. Use sync.Map or something similar to hold
// the callbacks
func OnStart(fn OnStartCb) {
	onStartListeners = append(onStartListeners, fn)
}
func notifyStart() {
	for _, f := range onStartListeners {
		f()
	}
}

type OnStopCb func()

var onStopListeners []OnStopCb

// FIXME: not thread safe. Use sync.Map or something similar to hold
// the callbacks
func OnStop(fn OnStopCb) {
	onStopListeners = append(onStopListeners, fn)
}

func notifyStop() {
	for _, f := range onStopListeners {
		f()
	}
}
