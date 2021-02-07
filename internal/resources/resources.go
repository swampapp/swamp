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

// load basic resources o they are ready for the app
func InitResources() {
	if err := os.MkdirAll(settings.DataDir(), 0755); err != nil {
		log.Error().Err(err)
	}

	f, err := resfs.Open("res.gresource")
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Error().Err(err)
		}
	}()

	rpath := filepath.Join(settings.DataDir(), "res.gresource")
	out, err := os.Create(rpath)
	if err != nil {
		panic(fmt.Errorf("error creating res.gresource: %v", err))
	}
	defer func() {
		if err = out.Close(); err != nil {
			log.Error().Err(err)
		}
	}()

	_, err = io.Copy(out, f)
	if err != nil {
		panic(fmt.Errorf("error copying res.gresource: %v", err))
	}

	res, err := gio.LoadGResource(rpath)
	if err != nil {
		log.Error().Err(err)
	}

	gio.RegisterGResource(res)
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
	case ".pdf", ".doc", ".xls", ".txt", ".md", ".rst", ".webarchive", ".html", ".pages", ".odf":
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
