package indexer

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rubiojr/rindex"
	"github.com/swampapp/swamp/internal/config"
	"github.com/swampapp/swamp/internal/resticsettings"
	"github.com/swampapp/swamp/internal/settings"
)

type Indexer struct {
	running          bool
	onStopListeners  []OnStopCb
	onStartListeners []OnStartCb
	mutex            sync.Mutex
}

type OnStopCb func()
type OnStartCb func()

var clientOnce, once sync.Once
var instance *Indexer
var socketClient *http.Client
var socketPath = filepath.Join(settings.DataDir(), "indexing.sock")
var log = zerolog.New(os.Stderr).With().Timestamp().Logger()

func New() *Indexer {
	i := &Indexer{}

	EnableDebugging(true)

	return i
}

func EnableDebugging(d bool) {
	if d {
		log = log.Level(zerolog.DebugLevel)
	} else {
		log = log.Level(zerolog.InfoLevel)
	}
}

func (i *Indexer) toggleState() {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	if IsRunning() && !i.running {
		i.running = true
		log.Print("indexer: running, notify start")
		i.notifyStart()
	} else if !IsRunning() && i.running {
		i.running = false
		log.Print("indexer: stopped, notify stop")
		i.notifyStop()
	}
}

func Daemon() *Indexer {
	once.Do(func() {
		instance = New()
		go func() {
			log.Print("indexer: starting swampd for the first time")
			instance.Start()
			ticker := time.NewTicker(60 * time.Minute)
			for range ticker.C {
				log.Print("indexer: trying to start swampd")
				instance.Start()
			}
		}()
		go func() {
			ticker := time.NewTicker(5 * time.Second)
			for range ticker.C {
				instance.toggleState()
			}
		}()
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

		i.notifyStart()

		for {
			if !resticsettings.FirstBoot() {
				log.Print("indexer: no first boot")
				break
			}
			time.Sleep(10 * time.Second)
			log.Print("indexer: waiting for first boot")
		}

		defer func() {
			i.notifyStop()
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
			i.notifyStop()
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

// FIXME: not thread safe. Use sync.Map or something similar to hold
// the callbacks
func (i *Indexer) OnStart(fn OnStartCb) {
	i.onStartListeners = append(i.onStartListeners, fn)
}

func (i *Indexer) notifyStart() {
	for _, f := range i.onStartListeners {
		f()
	}
}

// FIXME: not thread safe. Use sync.Map or something similar to hold
// the callbacks
func (i *Indexer) OnStop(fn OnStopCb) {
	i.onStopListeners = append(i.onStopListeners, fn)
}

func (i *Indexer) notifyStop() {
	for _, f := range i.onStopListeners {
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
		return rindex.IndexStats{}, err
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

func Pid() (int, error) {
	resp, err := Client().Get("http://localhost/pid")
	if err != nil {
		log.Print("error fetching stats")
		return -1, err
	}
	defer resp.Body.Close()

	var pid int
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return pid, err
	}

	return strconv.Atoi(string(b))
}

type ProcStats struct {
	RSS       uint64
	Duration  uint64
	StartTime time.Time
	CpuTime   uint64
}

func GetProcStats() (ProcStats, error) {
	var procStats ProcStats
	resp, err := Client().Get("http://localhost/procstats")
	if err != nil {
		log.Print("error fetching stats")
		return procStats, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return procStats, err
	}

	err = json.Unmarshal(b, &procStats)

	return procStats, err
}
