package config

import (
	"io/ioutil"
	"os"
	"sync"

	"github.com/swampapp/swamp/internal/logger"
	"github.com/swampapp/swamp/internal/paths"
	"gopkg.in/yaml.v2"
)

type Config struct {
	loaded        bool
	repositories  []Repository
	preferredRepo string
	darkMode      bool
	m             sync.Mutex
}

var prListeners []prListener

type Repository struct {
	Name string
	ID   string
}

type prListener func(string)

var instance Config
var once sync.Once

func (c Config) AddRepository(name, id string, preferred bool) {
	c.m.Lock()
	defer c.m.Unlock()

	c.repositories = append(c.repositories, Repository{ID: id, Name: name})

	if preferred {
		c.SetPreferredRepo(id)
	}

	c.Save()
}

func (c Config) Repositories() []Repository {
	return c.repositories
}

func (c Config) Save() error {
	d, err := yaml.Marshal(&c)
	if err != nil {
		logger.Fatal(err, "error marshalling configuration")
		return err
	}

	f, err := os.Create(paths.ConfigPath())
	if err != nil {
		logger.Error(err, "error creating config file")
		return err
	}

	_, err = f.Write(d)
	if err != nil {
		logger.Error(err, "error writing config file")
		return err
	}

	return nil
}

func Get() Config {
	if !instance.loaded {
		panic("configuration needs to be initialized first")
	}

	return instance
}

func Init() (Config, error) {
	var err error

	once.Do(func() {
		if Exists() {
			instance, err = Load()
			return
		}

		instance = Config{}
	})

	return instance, err
}

func Load() (Config, error) {
	c := Config{}

	if _, err := os.Stat(paths.ConfigPath()); err != nil {
		return c, err
	}

	f, err := ioutil.ReadFile(paths.ConfigPath())
	if err != nil {
		logger.Fatal(err, "error reading config")
		return c, err
	}

	err = yaml.Unmarshal([]byte(f), &c)
	if err != nil {
		logger.Fatal(err, "invalid config file format")
		return c, err
	}

	c.loaded = true

	return c, nil
}

func (c Config) PreferredRepo() string {
	return c.preferredRepo
}

func (c Config) SetPreferredRepo(id string) {
	c.m.Lock()
	defer c.m.Unlock()

	if c.preferredRepo == id {
		return
	}

	c.preferredRepo = id

	preferredChanged(id)

	logger.Debugf("setting preferred repo to %s", id)
	c.Save()
}

func AddPreferredRepoListener(l func(string)) {
	prListeners = append(prListeners, l)
}

func preferredChanged(id string) {
	for _, l := range prListeners {
		l(id)
	}
}

func (c Config) IsDarkMode() bool {
	return c.darkMode
}

func (c Config) SetDarkMode(mode bool) {
	c.darkMode = mode

	c.Save()
}

func Exists() bool {
	_, err := os.Stat(paths.ConfigPath())

	if err != nil && !os.IsNotExist(err) {
		panic(err)
	}

	return !os.IsNotExist(err)
}
