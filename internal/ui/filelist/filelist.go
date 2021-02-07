package filelist

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/blugelabs/bluge"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/rs/zerolog/log"
	"github.com/swampapp/swamp/internal/config"
	"github.com/swampapp/swamp/internal/downloader"
	"github.com/swampapp/swamp/internal/index"
	"github.com/swampapp/swamp/internal/queryparser"
	"github.com/swampapp/swamp/internal/resources"
	"github.com/swampapp/swamp/internal/status"
	"github.com/swampapp/swamp/internal/streamer"
	"github.com/swampapp/swamp/internal/tags"
	"github.com/swampapp/swamp/internal/ui/component"
	"github.com/swampapp/swamp/internal/ui/fileinfo"
	"github.com/swampapp/swamp/internal/ui/flview"
	"github.com/swampapp/swamp/internal/ui/tagger"
)

const maxResults = 500

var tagRegexp = regexp.MustCompile(`^tag:(?P<tag>[^\s]+)`)

type FileList struct {
	*component.Component
	*gtk.Box
	treeView         *flview.FLView
	searchEntry      *gtk.SearchEntry
	lastExportDir    string
	uniqueCBT        *gtk.CheckButton
	notDownloadedImg *gdk.Pixbuf
	downloadedImg    *gdk.Pixbuf
}

func New() *FileList {
	f := &FileList{Component: component.New("/ui/filelist")}

	f.uniqueCBT = f.GladeWidget("uniqueCBT").(*gtk.CheckButton)
	f.Box = f.GladeWidget("filelist").(*gtk.Box)
	f.searchEntry = f.GladeWidget("searchEntry").(*gtk.SearchEntry)
	f.searchEntry.SetCanFocus(true)
	f.setup()
	filelistSW := f.GladeWidget("filelistSW").(*gtk.ScrolledWindow)
	filelistSW.Add(f.treeView)

	config.AddPreferredRepoListener(func(rid string) {
		f.updateFileList("")
		searchEntry := f.GladeWidget("searchEntry").(*gtk.SearchEntry)
		searchEntry.SetText("")
	})

	f.GladeWidget("searchBtn").(*gtk.Button).Connect("clicked", func() {
		f.searchEntry.Emit("activate")
	})

	f.uniqueCBT.Connect("clicked", func() {
		t, _ := f.searchEntry.GetText()
		f.updateFileList(t)
	})

	return f
}

func (f *FileList) SetSearchText(text string) {
	f.searchEntry.SetText(text)
	f.searchEntry.SetCanDefault(true)
	f.searchEntry.GrabDefault()
	f.searchEntry.Emit("activate")
}

func (f *FileList) realize(tree *gtk.TreeView) {
	count := f.treeView.ItemCount()
	if count >= maxResults {
		status.SetRight(fmt.Sprintf("%d+ results", count))
	} else {
		status.SetRight(fmt.Sprintf("%d results", count))
	}
	f.searchEntry.GrabFocus()
}

func (f *FileList) secondButtonPressed(treeview *gtk.TreeView, btn *gdk.EventButton) {
	menu, _ := gtk.MenuNew()

	item, _ := gtk.MenuItemNew()
	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 16)
	box.SetHExpand(true)
	box.Add(resources.ScaledImage(24, 24, "action-download"))
	lbl, _ := gtk.LabelNew("Download")
	box.Add(lbl)
	item.Add(box)
	item.Connect("activate", func() bool {
		f.downloadSelected(false)
		return true
	})
	menu.Add(item)

	// Tag selecteds
	item, _ = gtk.MenuItemNew()
	box, _ = gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 16)
	box.SetHExpand(true)
	box.Add(resources.ScaledImage(24, 24, "action-tag"))
	lbl, _ = gtk.LabelNew("Tag")
	box.Add(lbl)
	item.Add(box)
	item.Connect("activate", func() bool {
		f.tagSelected()
		return true
	})
	menu.Add(item)

	// Download and open
	item, _ = gtk.MenuItemNew()
	box, _ = gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 16)
	box.SetHExpand(true)
	box.Add(resources.ScaledImage(24, 24, "action-open"))
	lbl, _ = gtk.LabelNew("Open")
	box.Add(lbl)
	item.Add(box)
	item.Connect("activate", func() bool {
		f.downloadSelected(true)
		return true
	})
	menu.Add(item)

	// Copy BHash
	item, _ = gtk.MenuItemNew()
	box, _ = gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 16)
	box.SetHExpand(true)
	box.Add(resources.ScaledImage(24, 24, "action-copy"))
	lbl, _ = gtk.LabelNew("Copy BHash")
	box.Add(lbl)
	item.Add(box)
	item.Connect("activate", func() bool {
		f.copyBHash(treeview)
		return true
	})
	menu.Add(item)

	// Find dupes
	item, _ = gtk.MenuItemNew()
	box, _ = gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 16)
	box.SetHExpand(true)
	box.Add(resources.ScaledImage(24, 24, "action-dupes"))
	lbl, _ = gtk.LabelNew("Find duplicates")
	box.Add(lbl)
	item.Add(box)
	item.Connect("activate", func() bool {
		f.findDuplicates()
		return true
	})
	menu.Add(item)

	if f.isStreamable(treeview, btn.X(), btn.Y()) {
		item, _ = gtk.MenuItemNew()
		box, _ = gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 16)
		box.SetHExpand(true)
		box.Add(resources.ScaledImage(24, 24, "action-stream"))
		lbl, _ = gtk.LabelNew("Stream")
		box.Add(lbl)
		item.Add(box)
		item.Connect("activate", func() bool {
			f.streamSelected()
			return true
		})
		menu.Add(item)
	}

	//Export
	item, _ = gtk.MenuItemNew()
	box, _ = gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 16)
	box.SetHExpand(true)
	box.Add(resources.ScaledImage(24, 24, "action-export"))
	lbl, _ = gtk.LabelNew("Export")
	box.Add(lbl)
	item.Add(box)
	item.Connect("activate", func() bool {
		f.exportFiles()
		return true
	})
	menu.Add(item)

	// Info
	item, _ = gtk.MenuItemNew()
	box, _ = gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 16)
	box.SetHExpand(true)
	box.Add(resources.ScaledImage(24, 24, "action-info"))
	lbl, _ = gtk.LabelNew("Info")
	box.Add(lbl)
	item.Add(box)
	item.Connect("activate", func() bool {
		f.showInfo()
		return true
	})
	menu.Add(item)

	menu.ShowAll()
	menu.PopupAtPointer(btn.Event)
}

func (f *FileList) isStreamable(tree *gtk.TreeView, x, y float64) bool {
	streamable := false

	fid, err := f.treeView.FileAt(int(y))
	if err != nil {
		log.Error().Err(err).Msgf("error retrieving file at row %d", int(y))
		return false
	}

	switch strings.ToLower(filepath.Ext(fid.Name)) {
	case ".mp4", ".avi", ".mkv", ".m4v", ".webm", ".mpeg", ".gif", ".mp3", ".wav", ".ogg", ".flac":
		streamable = true
	default:
	}

	return streamable
}

func (f *FileList) streamSelected() {
	for _, n := range f.treeView.SelectedFiles() {
		go func(file flview.File) {
			err := streamer.Stream(file.ID)
			if err != nil {
				status.Error("error streaming file")
			}
		}(n)
	}
}

func (f *FileList) downloadSelected(open bool) {
	files := f.treeView.SelectedFiles()
	if len(files) > 1 {
		status.Set("Downloading multiple files...")
	}

	for _, file := range files {
		d := downloader.Instance()
		if d.IsInProgress(file.ID) {
			status.Set("File is already being downloaded")
			return
		}
		if open {
			d.DownloadAndOpen(file.ID)
		} else {
			d.Download(file.ID)
		}
	}
}

func (f *FileList) findDuplicates() {
	files := f.treeView.SelectedFiles()
	for _, file := range files {
		doc, err := index.GetDocument(file.ID)
		if err != nil {
			status.Set("Error retrieving doc " + file.ID)
		}
		searchEntry := f.GladeWidget("searchEntry").(*gtk.SearchEntry)
		searchEntry.SetText("bhash:" + doc.BHash)
		searchEntry.Activate()
	}
}

func (f *FileList) copyBHash(tree *gtk.TreeView) {
	files := f.treeView.SelectedFiles()
	for _, file := range files {
		doc, err := index.GetDocument(file.ID)
		if err != nil {
			status.Set("Error retrieving doc " + file.ID)
		}
		clipboard, _ := gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD)
		clipboard.SetText(doc.BHash)
	}
}

func (f *FileList) setup() {
	f.treeView = flview.New()
	f.treeView.Connect("realize", f.realize)
	f.treeView.Connect("row-activated", func() bool {
		f.downloadSelected(false)
		return true
	})
	f.treeView.Connect("key-press-event", func(tree *gtk.TreeView, ev *gdk.Event) bool {
		kp := gdk.EventKeyNewFromEvent(ev)
		cntrlMask := uint(gdk.CONTROL_MASK)
		state := kp.State()
		cntrl := state&cntrlMask == cntrlMask
		switch kp.KeyVal() {
		case gdk.KEY_Return:
			f.downloadSelected(false)
			return true
		case gdk.KEY_d:
			if cntrl {
				f.downloadSelected(false)
				return true
			}
			return false
		case gdk.KEY_t:
			if cntrl {
				f.tagSelected()
				return true
			}
			return false
		case gdk.KEY_o:
			if cntrl {
				f.downloadSelected(true)
				return true
			}
			return false
		case gdk.KEY_s:
			if cntrl {
				f.streamSelected()
				return true
			}
			return false
		case gdk.KEY_i:
			if cntrl {
				f.showInfo()
				return true
			}
			return false
		case gdk.KEY_e:
			if cntrl {
				f.exportFiles()
				return true
			}
			return false
		default:
			return false
		}
	})

	f.treeView.Connect("button-press-event", func(tree *gtk.TreeView, ev *gdk.Event) bool {
		btn := gdk.EventButtonNewFromEvent(ev)
		switch btn.Button() {
		case gdk.BUTTON_SECONDARY:
			f.secondButtonPressed(f.treeView.TreeViewWidget(), btn)
			return false
		default:
			return false
		}
	})

	f.searchEntry.Connect("activate", func() {
		t, err := f.searchEntry.GetText()
		if err != nil {
			panic(err)
		}
		f.updateFileList(t)
	})
}

func (f *FileList) updateFileList(query string) {
	f.notDownloadedImg = resources.ImageForDoc("some.cloud")
	f.downloadedImg = resources.ImageForDoc("XXX")
	f.treeView.Clear()
	if !f.searchTags(query) {
		f.searchIndex(query)
	}
}

func (f *FileList) searchTags(query string) bool {
	if tagRegexp.Match([]byte(query)) {
		tname := ""
		match := tagRegexp.FindStringSubmatch(query)
		for i := range tagRegexp.SubexpNames() {
			if i > 0 && i <= len(match) {
				tname = match[i]
			}
		}
		log.Printf("searching for tag %s", tname)
		docs, _ := tags.GetDocuments(tname)
		for _, doc := range docs {
			if ok, _ := downloader.Instance().WasDownloaded(doc.ID); ok {
				f.treeView.AddRow(f.downloadedImg, doc.Name, doc.Path, doc.Size, doc.ID, doc.BHash)
			} else {
				f.treeView.AddRow(f.notDownloadedImg, doc.Name, doc.Path, doc.Size, doc.ID, doc.BHash)
			}
		}
		return true
	}

	return false
}

func (f *FileList) searchIndex(query string) {
	idx, err := index.Client()
	if err != nil {
		log.Err(err).Msg("error updating file list")
		return
	}

	filterDupes := f.uniqueCBT.GetActive()
	idCache := map[string]struct{}{}
	var fileID, filename, path, bhash string
	size := 0.0
	count := 0
	q, err := queryparser.ParseQuery(query)
	log.Debug().Msgf("searching for %s", q)
	if err != nil {
		status.Set("⚠️ invalid query: " + err.Error())
		return
	}

	_, err = idx.Search(q, func(field string, value []byte) bool {
		switch field {
		case "filename":
			filename = string(value)
		case "path":
			path = string(value)
		case "size":
			size, err = bluge.DecodeNumericFloat64(value)
			if err != nil {
				size = -1
			}
		case "_id":
			fileID = string(value)
		case "bhash":
			bhash = string(value)
		}

		return true
	},
		func() bool {
			_, found := idCache[bhash]
			if filterDupes && found {
				return true
			}
			idCache[bhash] = struct{}{}
			// FIXME: this is quite expensive with a large number of results
			if ok, _ := downloader.Instance().WasDownloaded(fileID); ok {
				f.treeView.AddRow(f.downloadedImg, filename, path, fmt.Sprintf("%.0f", size), fileID, bhash)
			} else {
				f.treeView.AddRow(f.notDownloadedImg, filename, path, fmt.Sprintf("%.0f", size), fileID, bhash)
			}
			count++
			return count <= maxResults
		},
	)

	if count == maxResults {
		status.SetRight(fmt.Sprintf("%d+ results", count))
	} else {
		status.SetRight(fmt.Sprintf("%d results", count))
	}

	// error searching, maybe there's no index yet
	if err != nil {
		status.Set("⚠️ Error while searching: " + err.Error())
	} else {
		status.Set("")
	}
}

func (f *FileList) showInfo() {
	files := f.treeView.SelectedFiles()
	if len(files) == 0 {
		return
	}

	w := fileinfo.NewWindow(files[0])
	w.ShowAll()
}

func (f *FileList) exportFiles() {
	files := f.treeView.SelectedFiles()
	if len(files) == 0 {
		return
	}

	fc, _ := gtk.FileChooserNativeDialogNew("Open", nil, gtk.FILE_CHOOSER_ACTION_SELECT_FOLDER, "_Open", "_Cancel")
	if f.lastExportDir != "" {
		fc.SetCurrentFolder(f.lastExportDir)
	}
	response := fc.NativeDialog.Run()
	if gtk.ResponseType(response) == gtk.RESPONSE_ACCEPT {
		f.lastExportDir = fc.GetFilename()
		//status.Set("Exporting files to " + targetDir)
		for _, file := range files {
			d := downloader.Instance()
			if d.IsInProgress(file.ID) {
				status.Set("File is already being downloaded")
				return
			}
			d.DownloadAndExport(file.ID, file.Name, f.lastExportDir)
		}
	}
}

func (f *FileList) tagSelected() {
	files := f.treeView.SelectedFiles()
	var fid string
	// FIXME: we only support tagging the first one selected for now
	for _, file := range files {
		fid = file.ID
		break
	}

	tw, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	tw.Add(tagger.New(fid))
	tw.Connect("key-press-event", func(w *gtk.Window, ev *gdk.Event) bool {
		kp := gdk.EventKeyNewFromEvent(ev)
		switch kp.KeyVal() {
		case gdk.KEY_Escape:
			tw.Destroy()
			return true
		default:
			return false
		}
	})
	tw.ShowAll()
}
