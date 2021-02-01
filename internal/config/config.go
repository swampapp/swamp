package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/rs/zerolog/log"
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

var shareDir = filepath.Join(os.Getenv("HOME"), ".local/share/com.github.swampapp")

func AddRepository(name, id string, preferred bool) {
	m.Lock()
	instance.Repositories = append(instance.Repositories, Repository{ID: id, Name: name})
	m.Unlock()

	if preferred {
		SetPreferredRepo(id)
	}

	if err := os.MkdirAll(filepath.Join(RepositoriesDir(), id), 0755); err != nil {
		log.Error().Err(err)
	}
}

func RepositoriesDir() string {
	return filepath.Join(shareDir, "repositories")
}

func init() {
	if err := os.MkdirAll(Dir(), 0755); err != nil {
		log.Error().Err(err)
	}

	if _, err := os.Stat(Path()); err == nil {
		Load()
	}
}

func PreferredRepoDir() string {
	return filepath.Join(RepositoriesDir(), PreferredRepo())
}

func RepoDirFor(name string) string {
	for _, r := range Repositories() {
		if r.Name == name {
			return filepath.Join(RepositoriesDir(), r.ID)
		}
	}

	return ""
}

func Repositories() []Repository {
	return instance.Repositories
}

func Save() {
	d, err := yaml.Marshal(&instance)
	if err != nil {
		log.Fatal().Err(err).Msgf("error: %v", err)
	}

	f, err := os.Create(Path())
	if err != nil {
		log.Error().Err(err).Msg("error creating config file")
	}

	_, err = f.Write(d)
	if err != nil {
		log.Error().Err(err).Msg("error writing config file")
	}
}

func Load() {
	f, err := ioutil.ReadFile(Path())
	if err != nil {
		log.Fatal().Err(err).Msg("error reading config")
	}

	err = yaml.Unmarshal([]byte(f), &instance)
	if err != nil {
		log.Fatal().Err(err).Msg("invalid config file format")
	}

	for _, repo := range instance.Repositories {
		if err := os.MkdirAll(filepath.Join(RepositoriesDir(), repo.ID), 0755); err != nil {
			log.Error().Err(err)
		}
	}
}

func Dir() string {
	return filepath.Join(os.Getenv("HOME"), ".config/com.github.swampapp")
}

func Path() string {
	return filepath.Join(os.Getenv("HOME"), ".config/com.github.swampapp", "config.yaml")
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

	log.Debug().Msgf("setting preferred repo to %s", id)
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
