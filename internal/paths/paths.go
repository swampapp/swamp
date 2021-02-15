package paths

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/swampapp/swamp/internal/logger"
)

var shareDir = filepath.Join(os.Getenv("HOME"), ".local/share/com.github.swampapp")

func init() {
	var err error

	if runtime.GOOS != "linux" {
		panic("OS currently not supported by swamp")
	}

	if err = os.MkdirAll(RepositoriesDir(), 0755); err != nil {
		logger.Error(err, "error creating repositories directory")
	}

	if err = os.MkdirAll(ConfigDir(), 0755); err != nil {
		logger.Error(err, "error creating configuration directory")
	}

	if err = os.MkdirAll(DownloadsDir(), 0755); err != nil {
		logger.Error(err, "error creating downloads directory")
	}

	if err != nil {
		panic(err)
	}
}

func DownloadsDir() string {
	return filepath.Join(shareDir, "downloads")
}

func DataDir() string {
	return shareDir
}

func ConfigDir() string {
	return filepath.Join(os.Getenv("HOME"), ".config/com.github.swampapp")
}

func RepositoriesDir() string {
	return filepath.Join(shareDir, "repositories")
}

func ConfigPath() string {
	return filepath.Join(os.Getenv("HOME"), ".config/com.github.swampapp", "config.yaml")
}
