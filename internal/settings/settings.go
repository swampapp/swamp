package settings

import (
	"os"
	"path/filepath"

	"github.com/swampapp/swamp/internal/config"
)

var shareDir = filepath.Join(os.Getenv("HOME"), ".local/share/com.github.swampapp")
var darkMode = false

func IsDarkMode() bool {
	return darkMode
}

func SetDarkMode(mode bool) {
	darkMode = mode
}

func init() {
	os.MkdirAll(config.RepositoriesDir(), 0755)
}

func Repository() string {
	return os.Getenv("RESTIC_REPOSITORY")
}

func Password() string {
	return os.Getenv("RESTIC_PASSWORD")
}

func IndexPath() string {
	return filepath.Join(IndexDir(), "swamp.bluge")
}

func IndexDir() string {
	return filepath.Join(RepoDir(), "index")
}

func RepoDir() string {
	return filepath.Join(config.RepositoriesDir(), config.PreferredRepo())
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

func BinDir() string {
	return filepath.Join(DataDir(), "bin")
}

func IconsDir() string {
	return filepath.Join(os.Getenv("HOME"), ".local/share/icons")
}

func AppsDir() string {
	return filepath.Join(os.Getenv("HOME"), ".local/share/applications")
}
