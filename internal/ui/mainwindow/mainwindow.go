package mainwindow

import (
	"fmt"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
	"github.com/swampapp/swamp/internal/config"
	"github.com/swampapp/swamp/internal/downloader"
	"github.com/swampapp/swamp/internal/eventbus"
	indexerd "github.com/swampapp/swamp/internal/indexer"
	"github.com/swampapp/swamp/internal/logger"
	"github.com/swampapp/swamp/internal/resources"
	"github.com/swampapp/swamp/internal/status"
	"github.com/swampapp/swamp/internal/streamer"
	"github.com/swampapp/swamp/internal/ui/appmenu"
	"github.com/swampapp/swamp/internal/ui/component"
	"github.com/swampapp/swamp/internal/ui/downloadlist"
	"github.com/swampapp/swamp/internal/ui/filelist"
	"github.com/swampapp/swamp/internal/ui/indexer"
	"github.com/swampapp/swamp/internal/ui/inprogresslist"
	settingsui "github.com/swampapp/swamp/internal/ui/settings"
	"github.com/swampapp/swamp/internal/ui/taglist"
)

type MainWindow struct {
	*component.Component
	gtk.ApplicationWindow
	appMenu        *appmenu.AppMenu
	fileList       *filelist.FileList
	downloadList   *downloadlist.DownloadList
	inprogressList *inprogresslist.InProgressList
	tagList        *taglist.TagList
	paned          *gtk.Paned
	searchText     string
	indexerUI      *indexer.Indexer
}

func New(a *gtk.Application) (*MainWindow, error) {
	w, err := gtk.ApplicationWindowNew(a)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create window")
	}

	// Detect if a dark theme is being used
	ctx, _ := w.GetStyleContext()
	color := ctx.GetColor(gtk.STATE_FLAG_NORMAL)
	rgba := color.Floats()
	luminace := (0.2126*rgba[0] + 0.7152*rgba[1] + 0.0722*rgba[2])
	if luminace > 0.5 {
		config.Get().SetDarkMode(true)
	}

	mw := &MainWindow{
		Component:         component.New("/ui/mainwindow"),
		ApplicationWindow: *w,
		appMenu:           appmenu.New(),
		downloadList:      downloadlist.New(),
		inprogressList:    inprogresslist.New(),
		tagList:           taglist.New(),
		fileList:          filelist.New(),
		indexerUI:         indexer.New(),
	}

	resources.LoadImages()

	mw.paned = mw.GladeWidget("content_panel").(*gtk.Paned)

	eventbus.ListenTo(
		appmenu.SelectionChangedEvent,
		func(evt *eventbus.Event) {
			sel := evt.Data.(string)
			mw.SetMainPanel(sel)
		},
	)

	w.SetTitle("Swamp")
	w.SetDefaultSize(1024, 600)
	w.SetIcon(resources.Pixbuf("appicon"))
	w.Connect("key-press-event", func(tree *gtk.ApplicationWindow, ev *gdk.Event) bool {
		kp := gdk.EventKeyNewFromEvent(ev)
		cntrlMask := uint(gdk.CONTROL_MASK)
		state := kp.State()
		cntrl := state&cntrlMask == cntrlMask
		switch kp.KeyVal() {
		case gdk.KEY_q:
			if cntrl {
				a.Quit()
				return true
			}
			return false
		case gdk.KEY_f:
			if cntrl {
				mw.appMenu.SelectPath("0")
				return true
			}
			return false
		default:
			return false
		}
	})

	container := mw.GladeWidget("container").(*gtk.Box)
	mw.Container.Add(container)

	pane := mw.GladeWidget("content_panel").(*gtk.Paned)
	if err != nil {
		panic(err)
	}

	// App Menu
	pane.Add1(mw.appMenu.Widget())

	// File List
	pane.Add2(mw.fileList)
	pane.SetPosition(230)

	eventbus.ListenTo(
		taglist.TagSelectedEvent,
		func(evt *eventbus.Event) {
			tag := evt.Data.(string)
			mw.searchText = "tag:" + tag
			mw.appMenu.SelectPath("0")
			mw.searchText = ""
		},
	)

	mw.StopDownloading()
	mw.StopIndexing()
	mw.appMenu.SelectPath("0")

	eventbus.ListenTo(
		streamer.StreamingStarted,
		func(*eventbus.Event) {
			mw.StartDownloading()
		},
	)

	eventbus.ListenTo(
		streamer.StreamingStopped,
		func(*eventbus.Event) {
			mw.StopDownloading()
		},
	)

	eventbus.ListenTo(
		indexerd.IndexingStartedEvent,
		func(*eventbus.Event) {
			mw.StartIndexing()
		},
	)

	eventbus.ListenTo(
		indexerd.IndexingStoppedEvent,
		func(*eventbus.Event) {
			mw.StopIndexing()
		},
	)

	eventbus.ListenTo(
		status.SetEvent,
		func(evt *eventbus.Event) {
			mw.SetStatus(evt.Data.(string))
		},
	)

	eventbus.ListenTo(
		status.SetRightEvent,
		func(evt *eventbus.Event) {
			mw.SetStatusRight(evt.Data.(string))
		},
	)

	eventbus.ListenTo(
		downloader.DownloadStartedEvent,
		mw.downloadStarted,
	)

	eventbus.ListenTo(
		downloader.DownloadFailedEvent,
		mw.downloadFailed,
	)

	eventbus.ListenTo(
		downloader.DownloadFinishedEvent,
		mw.downloadFinished,
	)

	eventbus.ListenTo(
		downloader.QueueEmptyEvent,
		mw.downloadQueueEmpty,
	)

	return mw, nil
}

func (w *MainWindow) SetMainPanel(t string) {
	logger.Print("changing main pane to ", t)

	child, _ := w.paned.GetChild2()
	if child != nil {
		child.ToWidget().Hide()
		w.paned.Remove(child)
	}

	if t != "Search" {
		w.searchText = ""
	}

	switch t {
	case "Search":
		w.appMenu.SelectPath("0")
		w.paned.Add2(w.fileList)
		if w.searchText != "" {
			w.fileList.SetSearchText(w.searchText)
		}
	case "Tags":
		w.paned.Add2(w.tagList)
	case "Downloaded":
		w.paned.Add2(w.downloadList)
	case "In Progress":
		w.paned.Add2(w.inprogressList)
	case "Indexer":
		w.paned.Add2(w.indexerUI)
	case "Settings":
		w.paned.Add2(settingsui.New())
	default:
		panic(fmt.Errorf("panel '%s' not implemented", t))
	}
	w.paned.ShowAll()
}

func (w *MainWindow) StatusClear() {
	w.SetStatus("")
}

func (w *MainWindow) StatusWarn(msg string) {
	w.SetStatus(fmt.Sprintf("⚠️ %s", msg))
}

func (w *MainWindow) StatusError(msg string) {
	w.SetStatus(fmt.Sprintf("🛑 %s", msg))
}

func (w *MainWindow) SetStatus(text string) {
	glib.IdleAdd(func() {
		statusLabel := w.GladeWidget("statusLabel").(*gtk.Label)
		statusLabel.SetText(text)
	})
}

func (w *MainWindow) SetStatusRight(text string) {
	glib.IdleAdd(func() {
		statusLabel := w.GladeWidget("statusLabelRight").(*gtk.Label)
		statusLabel.SetText(text)
	})
}

func (w *MainWindow) StartDownloading() {
	glib.IdleAdd(func() {
		img := w.GladeWidget("downloadingIMG").(*gtk.Image)
		resources.UpdateImageFromResource(img, "status-downloading")
		img.SetTooltipText("Indexing documents")
		img.Show()
	})
}

func (w *MainWindow) StopDownloading() {
	glib.IdleAdd(func() {
		img := w.GladeWidget("downloadingIMG").(*gtk.Image)
		resources.UpdateImageFromResource(img, "ui/statusbar/finished")
		img.SetTooltipText("Indexing documents")
		img.Show()
	})
}

func (w *MainWindow) FailedDownloading() {
	glib.IdleAdd(func() {
		img := w.GladeWidget("downloadingIMG").(*gtk.Image)
		resources.UpdateImageFromResource(img, "ui/statusbar/download-failed")
		img.SetTooltipText("Indexing documents")
		img.Show()
	})
}

func (w *MainWindow) StartIndexing() {
	glib.IdleAdd(func() {
		img := w.GladeWidget("indexingIMG").(*gtk.Image)
		resources.UpdateImageFromResource(img, "status-indexing")
		img.SetTooltipText("Indexing documents")
		img.Show()
	})
}

func (w *MainWindow) StopIndexing() {
	glib.IdleAdd(func() {
		img := w.GladeWidget("indexingIMG").(*gtk.Image)
		resources.UpdateImageFromResource(img, "ui/statusbar/indexed")
		img.SetTooltipText("Index ready")
	})
}

func (w *MainWindow) downloadStarted(evt *eventbus.Event) {
	w.StartDownloading()
	w.SetStatus("Downloading files...")
}

func (w *MainWindow) downloadFailed(evt *eventbus.Event) {
	w.FailedDownloading()
	w.SetStatus("Downloading failed!")
}

func (w *MainWindow) downloadFinished(evt *eventbus.Event) {
	w.StopDownloading()
}

func (w *MainWindow) downloadQueueEmpty(evt *eventbus.Event) {
	w.StopDownloading()
}
