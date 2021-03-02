package indexer

import (
	"context"
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

	"github.com/rubiojr/rindex"
	"github.com/swampapp/swamp/internal/config"
	"github.com/swampapp/swamp/internal/credentials"
	"github.com/swampapp/swamp/internal/eventbus"
	"github.com/swampapp/swamp/internal/logger"
	"github.com/swampapp/swamp/internal/paths"
)

var IndexingStartedEvent = "indexer.indexing_started"
var IndexingStoppedEvent = "indexer.indexing_stopped"

type Indexer struct {
	running bool
	mutex   sync.Mutex
}

var clientOnce, once sync.Once
var instance *Indexer
var socketClient *http.Client
var socketPath = filepath.Join(paths.DataDir(), "indexing.sock")

func New() *Indexer {
	return &Indexer{}
}

func (i *Indexer) toggleState() {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	if IsRunning() && !i.running {
		i.running = true
		logger.Print("indexer: running, notify start")
		eventbus.Emit(context.Background(), IndexingStartedEvent, nil)
	} else if !IsRunning() && i.running {
		i.running = false
		logger.Print("indexer: stopped, notify stop")
		eventbus.Emit(context.Background(), IndexingStoppedEvent, nil)
	}
}

func Daemon() *Indexer {
	once.Do(func() {
		eventbus.RegisterTopics(IndexingStartedEvent, IndexingStoppedEvent)
		instance = New()
		go func() {
			logger.Print("indexer: starting swampd for the first time")
			instance.Start()
			ticker := time.NewTicker(60 * time.Minute)
			for range ticker.C {
				logger.Print("indexer: trying to start swampd")
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
		if !config.Exists() {
			logger.Warn("indexer: configuration does not exist")
			return
		}

		if IsRunning() {
			logger.Print("indexer: already running, skiping")
			return
		}

		logger.Print("indexer: STARTED the indexing goroutine")
		prepo := config.Get().PreferredRepo()
		rs := credentials.New(prepo)

		for {
			if !credentials.FirstBoot() {
				logger.Print("indexer: no first boot")
				break
			}
			time.Sleep(10 * time.Second)
			logger.Print("indexer: waiting for first boot")
		}

		defer func() {
			logger.Print("indexer: stopped swampd")
		}()

		logger.Print("indexer: starting swampd")
		bin, err := exec.LookPath("swampd")
		if err != nil {
			logger.Error(err, "error finding swampd executable path")
			return
		}

		args := []string{"--debug", "--index-path", currentIndexPath(), "index"}
		logger.Print("swampd command: ", args)
		cmd := exec.Command(bin, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, "RESTIC_REPOSITORY="+rs.Repository)
		cmd.Env = append(cmd.Env, "RESTIC_PASSWORD="+rs.Password)
		cmd.Env = append(cmd.Env, "AWS_ACCESS_KEY="+rs.Var1)
		cmd.Env = append(cmd.Env, "AWS_SECRET_ACCESS_KEY="+rs.Var2)
		err = cmd.Run()
		if err != nil {
			logger.Error(err, "indexer: swampd error")
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
		logger.Error(err, "unhandled error reading body")
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
		logger.Error(err, "unhandled error reading body")
	}
	return string(b) == "pong"
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
		logger.Print("error fetching stats")
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
		logger.Print("error fetching stats")
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
		logger.Print("error fetching stats")
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

func currentIndexPath() string {
	pr := config.Get().PreferredRepo()
	rd := paths.RepositoriesDir()

	return filepath.Join(rd, pr, "index", "swamp.bluge")
}
