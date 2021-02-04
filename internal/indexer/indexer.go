package indexer

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rubiojr/rindex"
	"github.com/swampapp/swamp/internal/config"
	"github.com/swampapp/swamp/internal/resticsettings"
	"github.com/swampapp/swamp/internal/settings"
)

type Indexer struct {
}

var clientOnce, once sync.Once
var instance *Indexer
var socketClient *http.Client
var socketPath = filepath.Join(settings.DataDir(), "indexing.sock")
var mutex sync.Mutex
var running bool
var log = zerolog.New(os.Stderr).With().Timestamp().Logger()

func init() {
	EnableDebugging(false)

	go func() {
		ticker := time.NewTicker(30 * time.Minute)
		for range ticker.C {
			log.Print("indexer: trying to start swampd")
			Daemon().Start()
		}
	}()
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		for range ticker.C {
			toggleState()
		}
	}()
}

func EnableDebugging(d bool) {
	if d {
		log = log.Level(zerolog.DebugLevel)
	} else {
		log = log.Level(zerolog.InfoLevel)
	}
}

func toggleState() {
	mutex.Lock()
	defer mutex.Unlock()

	if IsRunning() && !running {
		running = true
		log.Print("indexer: running, notify start")
		notifyStart()
	} else if !IsRunning() && running {
		running = false
		log.Print("indexer: stopped, notify stop")
		notifyStop()
	}
}

func Daemon() *Indexer {
	once.Do(func() {
		instance = &Indexer{}
	})

	return instance
}

func (i *Indexer) Start() {
	go func() {
		if IsRunning() {
			log.Print("indexer: already running, skiping")
			return
		}

		log.Print("indexer: STARTED the indexing goroutine")
		prepo := config.PreferredRepo()
		rs := resticsettings.New(prepo)

		notifyStart()

		for {
			if !resticsettings.FirstBoot() {
				log.Print("indexer: no first boot")
				break
			}
			time.Sleep(10 * time.Second)
			log.Print("indexer: waiting for first boot")
		}

		defer func() {
			notifyStop()
			log.Print("indexer: stopped swampd")
		}()

		log.Print("indexer: starting swampd")
		bin, err := exec.LookPath("swampd")
		if err != nil {
			log.Error().Err(err).Msg("error finding swampd executable path")
			return
		}

		log.Printf("indexer: %s %s %s %s", bin, "--index-path", settings.IndexPath(), "index")
		cmd := exec.Command(bin, "--debug", "--index-path", settings.IndexPath(), "index")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, "RESTIC_REPOSITORY="+rs.Repository)
		cmd.Env = append(cmd.Env, "RESTIC_PASSWORD="+rs.Password)
		cmd.Env = append(cmd.Env, "AWS_ACCESS_KEY="+rs.Var1)
		cmd.Env = append(cmd.Env, "AWS_SECRET_ACCESS_KEY="+rs.Var2)
		err = cmd.Run()
		if err != nil {
			log.Error().Err(err).Msgf("indexer: swampd error")
		}
	}()
}

func (i *Indexer) Stop() error {
	resp, err := Client().Post("http://localhost/kill", "text/plain", nil)
	if err != nil {
		return err
	}
	//nolint
	defer resp.Body.Close()

	_, err = io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Msg("unhandled error reading body")
		if !IsRunning() {
			notifyStop()
		}
	}
	return err
}

func IsRunning() bool {
	resp, err := Client().Get("http://localhost/ping")
	if err != nil {
		return false
	}
	//nolint
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Msg("unhandled error reading body")
	}
	return string(b) == "pong"
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

func Client() *http.Client {
	clientOnce.Do(func() {
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

func SocketPath() string {
	return socketPath
}

func Stats() (rindex.IndexStats, error) {
	resp, err := Client().Get("http://localhost/stats")
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
