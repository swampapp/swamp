package inprogresslist

import (
	"github.com/gotk3/gotk3/gtk"
	"github.com/swampapp/swamp/internal/downloader"
	"github.com/swampapp/swamp/internal/resources"
	"github.com/swampapp/swamp/internal/ui/component"
)

type InProgressList struct {
	*component.Component
	*gtk.Box
	listBox *gtk.ListBox
}

func New() *InProgressList {
	i := &InProgressList{Component: component.New("/ui/inprogresslist")}
	i.Box = i.GladeWidget("container").(*gtk.Box)
	filelistSW := i.GladeWidget("queuedlistSW").(*gtk.ScrolledWindow)
	i.listBox, _ = gtk.ListBoxNew()

	i.listBox.SetSelectionMode(gtk.SELECTION_SINGLE)
	i.updateFileList()
	i.listBox.Connect("realize", i.isShown)

	filelistSW.Add(i.listBox)

	return i
}

func (i *InProgressList) addFileRow(filename string) error {
	row, err := gtk.ListBoxRowNew()
	if err != nil {
		panic(err)
	}
	row.SetMarginTop(8)

	hbox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 12)
	imageLoading := resources.Image("downloading-animation")
	hbox.PackStart(imageLoading, false, false, 2)
	hbox.SetVExpand(false)
	hbox.SetSpacing(4)

	label, _ := gtk.LabelNew(filename)
	hbox.PackStart(label, false, false, 2)

	row.Add(hbox)
	row.ShowAll()
	i.listBox.Add(row)

	return err
}

func (i *InProgressList) isShown(tree *gtk.TreeView) {
	i.updateFileList()
}

func (i *InProgressList) updateFileList() {
	//TODO: use leveldb to speed things up without walking the filesystem
	i.listBox.BindModel(nil, nil)

	for _, doc := range downloader.Instance().DownloadsInProgress() {
		i.addFileRow(doc.Name)
	}
}
