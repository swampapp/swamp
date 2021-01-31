package resources

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"embed"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gio"
	"github.com/gotk3/gotk3/gtk"
	"github.com/rs/zerolog/log"
	"github.com/swampapp/swamp/internal/settings"
)

var imageCloud, imageCompressed, imageOther, imageImage, imageAudio, imageVideo, imageDoc *gdk.Pixbuf

//go:embed res.gresource
var resfs embed.FS

//go:embed swamp.desktop
var dotDesktop embed.FS

//go:embed swampapp.png
var iconfs embed.FS

// load basic resources o they are ready for the app
func InitResources() {
	os.MkdirAll(settings.DataDir(), 0755)

	f, err := resfs.Open("res.gresource")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	rpath := filepath.Join(settings.DataDir(), "res.gresource")
	out, err := os.Create(rpath)
	defer out.Close()
	_, err = io.Copy(out, f)
	if err != nil {
		panic(err)
	}
	out.Close()

	res, err := gio.LoadGResource(rpath)
	gio.RegisterGResource(res)

	//copyIcon()
	//copyDesktop()
}

func copyDesktop() {
	rpath := filepath.Join(settings.AppsDir(), "com.github.swampapp.desktop")
	os.MkdirAll(settings.BinDir(), 0755)
	out, err := os.Create(rpath)
	if err != nil {
		log.Error().Err(err).Msg("error creating .desktop file")
		return
	}
	defer out.Close()

	f, err := dotDesktop.Open("swamp.desktop")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	_, err = io.Copy(out, f)
	if err != nil {
		log.Error().Err(err).Msg("could not copy desktop file")
		return
	}
}

func copyIcon() {
	rpath := filepath.Join(settings.IconsDir(), "swampapp.png")
	os.MkdirAll(settings.BinDir(), 0755)
	out, err := os.Create(rpath)
	if err != nil {
		log.Error().Err(err).Msg("error creating icon")
		return
	}
	defer out.Close()

	f, err := iconfs.Open("swampapp.png")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	_, err = io.Copy(out, f)
	if err != nil {
		log.Error().Err(err).Msg("could not copy swampd icon")
	}
}

func LoadImages() {
	imageAudio = Pixbuf("type-audio")
	imageVideo = Pixbuf("type-video")
	imageDoc = Pixbuf("type-document")
	imageImage = Pixbuf("type-image")
	imageCompressed = Pixbuf("type-compressed")
	imageOther = Pixbuf("save")
	imageCloud = Pixbuf("doc")
}

func CSS() *gtk.CssProvider {
	css, _ := gtk.CssProviderNew()
	css.LoadFromResource("/ui/stylesheet")
	return css
}

func ImageForDoc(name string) *gdk.Pixbuf {
	ext := filepath.Ext(name)
	switch ext {
	case ".png", ".jpeg", ".jpg", ".tiff", ".gif", ".svg":
		return imageImage
	case ".avi", ".mp4", ".mkv", ".mov", ".webm":
		return imageVideo
	case ".mp3", ".flac", ".ogg", ".wav", ".m4p":
		return imageAudio
	case ".pdf", ".doc", ".xls", ".txt", ".md", ".rst":
		return imageDoc
	case ".zip", ".tar", ".tgz", ".rar", ".gz", ".bz2", ".xz":
		return imageCompressed
	case ".cloud":
		return imageCloud
	default:
		return imageOther
	}
}

// If this ever fails, we should crash hard
func Pixbuf(path string) *gdk.Pixbuf {
	// FIXME: hack, how do we detect a dark theme reliably?
	accent := "light"
	if settings.IsDarkMode() {
		accent = "dark"
	}
	rpath := fmt.Sprintf("/images/%s/%s", accent, path)
	img, err := gtk.ImageNewFromResource(rpath)
	if err != nil {
		panic(err)
	}
	return img.GetPixbuf()
}

func UpdateImageFromResource(img *gtk.Image, path string) {
	accent := "light"
	if settings.IsDarkMode() {
		accent = "dark"
	}
	rpath := fmt.Sprintf("/images/%s/%s", accent, path)
	img.SetFromResource(rpath)
}

func Image(path string) *gtk.Image {
	accent := "light"
	if settings.IsDarkMode() {
		accent = "dark"
	}
	rpath := fmt.Sprintf("/images/%s/%s", accent, path)
	img, err := gtk.ImageNewFromResource(rpath)
	if err != nil {
		panic(err)
	}
	return img
}

func ScaledPixbuf(width, height int, path string) *gdk.Pixbuf {
	p, _ := Pixbuf(path).ScaleSimple(width, height, gdk.INTERP_HYPER)
	return p
}

func ScaledImage(width, height int, path string) *gtk.Image {
	p, _ := Pixbuf(path).ScaleSimple(width, height, gdk.INTERP_HYPER)
	i, _ := gtk.ImageNewFromPixbuf(p)
	return i
}
