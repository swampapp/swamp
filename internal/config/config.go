package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/swampapp/swamp/internal/logger"
	"github.com/swampapp/swamp/internal/paths"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Repositories  []Repository
	PreferredRepo string
}

type Repository struct {
	Name string
	ID   string
}

var m sync.Mutex

type prListener func(string)

var prListeners []prListener

var instance = &Config{}

func AddRepository(name, id string, preferred bool) {
	m.Lock()
	instance.Repositories = append(instance.Repositories, Repository{ID: id, Name: name})
	m.Unlock()

	if preferred {
		SetPreferredRepo(id)
	}
}

func init() {
	if _, err := os.Stat(paths.ConfigPath()); err == nil {
		Load()
	}
}

func Repositories() []Repository {
	return instance.Repositories
}

func Save() {
	d, err := yaml.Marshal(&instance)
	if err != nil {
		logger.Fatal(err, "error marshalling configuration")
	}

	f, err := os.Create(paths.ConfigPath())
	if err != nil {
		logger.Error(err, "error creating config file")
	}

	_, err = f.Write(d)
	if err != nil {
		logger.Error(err, "error writing config file")
	}
}

func Load() {
	f, err := ioutil.ReadFile(paths.ConfigPath())
	if err != nil {
		logger.Fatal(err, "error reading config")
	}

	err = yaml.Unmarshal([]byte(f), &instance)
	if err != nil {
		logger.Fatal(err, "invalid config file format")
	}

	for _, repo := range instance.Repositories {
		if err := os.MkdirAll(filepath.Join(paths.RepositoriesDir(), repo.ID), 0755); err != nil {
			logger.Errorf(err, "error creaating repository %s directory", repo.ID)
		}
	}
}

func PreferredRepo() string {
	return instance.PreferredRepo
}

func SetPreferredRepo(id string) {
	m.Lock()
	defer m.Unlock()
	if instance.PreferredRepo == id {
		return
	}
	instance.PreferredRepo = id
	preferredChanged(id)

	logger.Debugf("setting preferred repo to %s", id)
	Save()
}

func AddPreferredRepoListener(l func(string)) {
	prListeners = append(prListeners, l)
}

func preferredChanged(id string) {
	for _, l := range prListeners {
		l(id)
	}
}

var darkMode = false

func IsDarkMode() bool {
	return darkMode
}

func SetDarkMode(mode bool) {
	darkMode = mode
}
