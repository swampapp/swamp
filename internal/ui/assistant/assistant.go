package assistant

import (
	"os"
	"strings"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/rubiojr/rapi"
	"github.com/swampapp/swamp/internal/config"
	"github.com/swampapp/swamp/internal/credentials"
	"github.com/swampapp/swamp/internal/logger"
	"github.com/swampapp/swamp/internal/ui/component"
)

var callback func()

type Assistant struct {
	*gtk.Assistant
	*component.Component
}

func New() *Assistant {
	a := &Assistant{Component: component.New("/ui/assistant")}
	a.Assistant = a.GladeWidget("container").(*gtk.Assistant)
	l, _ := gtk.LabelNew("")
	a.AddActionWidget(l)

	// Combobox
	store, _ := gtk.ListStoreNew(glib.TYPE_STRING)
	addRow(store, "Local or Rest")
	addRow(store, "S3")
	combo := a.GladeWidget("assistantCombo").(*gtk.ComboBox)
	combo.SetModel(store)
	rtext, _ := gtk.CellRendererTextNew()
	combo.PackStart(rtext, false)
	combo.AddAttribute(rtext, "text", 0)
	combo.Connect("changed", func(d interface{}) {
		iter, _ := combo.GetActiveIter()
		value, _ := store.GetValue(iter, 0)
		rtype, _ := value.GetString()
		b, _ := a.GladeWidget("assistantExtraBox").(*gtk.Box)
		if rtype == "S3" {
			b.Show()
		} else {
			b.Hide()
		}
	})

	var rs *credentials.Credentials
	page2 := a.GladeWidget("assistantPage2").(*gtk.Box)
	btn := a.GladeWidget("testSettingsBTN").(*gtk.Button)
	var repoID, repoName, ruri, rpass, var1, var2 string
	btn.Connect("clicked", func() {
		lbl := a.GladeWidget("settingsTestLabel").(*gtk.Label)
		uri := a.GladeWidget("repoURIEntry").(*gtk.Entry)
		pass := a.GladeWidget("repoPasswordEntry").(*gtk.Entry)
		uri.Connect("changed", func() {
			lbl.SetText("")
		})
		pass.Connect("changed", func() {
			lbl.SetText("")
		})
		ruri = a.getEntryText("repoURIEntry")
		rpass = a.getEntryText("repoPasswordEntry")
		var1 = a.getEntryText("accessKeyEntry")
		var2 = a.getEntryText("secretAccessKeyEntry")
		repoName = a.getEntryText("repoNameEntry")
		dopts := rapi.DefaultOptions
		dopts.Repo = ruri
		dopts.Password = rpass
		os.Unsetenv("RESTIC_REPOSITORY")
		os.Unsetenv("RESTIC_PASSWORD")
		os.Setenv("AWS_ACCESS_KEY", var1)
		os.Setenv("AWS_SECRET_ACCESS_KEY", var2)
		go func() {
			glib.IdleAdd(func() {
				lbl.SetText("🕐 one second, checking credentials...")
			})
			repo, err := rapi.OpenRepository(dopts)
			if err != nil {
				logger.Error(err, "invalid restic credentials")
				glib.IdleAdd(func() {
					lbl.SetText("⚠️ Invalid repository settings")
				})
			} else {
				glib.IdleAdd(func() {
					repoID = repo.Config().ID
					rs = credentials.New(repoID)
					rs.Password = rpass
					rs.Repository = ruri
					rs.Var1 = var1
					rs.Var2 = var2
					err := rs.Save()
					if err != nil {
						msg := "could not save credentials in the keyring"
						logger.Error(err, msg)
						lbl.SetText("⚠️ " + msg)
						return
					}
					lbl.SetMarkup("👍 It worked! Click <b>Next</b> to finish the configuration.")
					logger.Print("credentials saved")
					a.SetPageComplete(page2, true)
				})
			}
		}()
	})

	a.Connect("cancel", func(widget *gtk.Assistant) {
		os.Exit(1)
	})

	a.Connect("apply", func(widget *gtk.Assistant) {
		config.Get().AddRepository(repoName, repoID, true)
		widget.Hide()
		if callback != nil {
			callback()
		}
	})

	return a
}

func (a *Assistant) WhenDone(cb func()) {
	callback = cb
}

// Append a row to the list store for the tree view
func addRow(listStore *gtk.ListStore, text string) {
	// Get an iterator for a new row at the end of the list store
	iter := listStore.Append()

	// Set the contents of the list store row that the iterator represents
	err := listStore.Set(iter,
		[]int{0},
		[]interface{}{text})
	if err != nil {
		logger.Print("Unable to add row")
		panic(err)
	}
}

func (a *Assistant) getEntryText(name string) string {
	e := a.GladeWidget(name).(*gtk.Entry)
	t, err := e.GetText()
	if err != nil {
		panic(err)
	}
	return strings.TrimSpace(t)
}
