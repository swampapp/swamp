package reposelector

import (
	"github.com/rs/zerolog/log"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/swampapp/swamp/internal/config"
	"github.com/swampapp/swamp/internal/ui/component"
)

type ColID int

const (
	COLUMN_NAME ColID = iota
	COLUMN_ID
)

type RepoSelector struct {
	*component.Component
	*gtk.ComboBox
	store *gtk.ListStore
}

func New() *RepoSelector {
	rs := &RepoSelector{}
	rs.store, _ = gtk.ListStoreNew(glib.TYPE_STRING, glib.TYPE_STRING)
	rs.ComboBox, _ = gtk.ComboBoxNew()
	rs.SetModel(rs.store)
	rs.populate()
	rtext, _ := gtk.CellRendererTextNew()
	rs.PackStart(rtext, false)
	rs.AddAttribute(rtext, "text", 0)
	rs.SetTooltipText("Select the Restic repository to use")
	rs.Connect("realize", func(d interface{}) bool {
		rs.repoChanged()
		return false
	})

	rs.Connect("changed", func(d interface{}) bool {
		iter, err := rs.GetActiveIter()
		if err != nil {
			return false
		}
		value, _ := rs.store.GetValue(iter, 1)
		repo, _ := value.GetString()
		config.SetPreferredRepo(repo)

		return true
	})

	return rs
}

func (rs *RepoSelector) populate() {
	rs.store.Clear()
	for _, repo := range config.Repositories() {
		iter := rs.store.Prepend()
		err := rs.store.Set(iter,
			[]int{0, 1},
			[]interface{}{repo.Name, repo.ID})
		if err != nil {
			log.Print("Unable to add row")
			panic(err)
		}
	}
}

func (rs *RepoSelector) repoChanged() {
	rs.store.Clear()
	for _, repo := range config.Repositories() {
		iter := rs.store.Prepend()
		err := rs.store.Set(iter,
			[]int{0, 1},
			[]interface{}{repo.Name, repo.ID})
		if err != nil {
			log.Print("Unable to add row")
			panic(err)
		}
		if config.PreferredRepo() == repo.ID {
			rs.SetActiveIter(iter)
		}
	}
}
