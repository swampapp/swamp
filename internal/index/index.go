package index

import (
	"context"
	"fmt"

	"github.com/blugelabs/bluge"
	"github.com/rubiojr/rindex"
	"github.com/swampapp/swamp/internal/config"
	"github.com/swampapp/swamp/internal/keyring"
	"github.com/swampapp/swamp/internal/logger"
)

type Document struct {
	Name  string
	Path  string
	ID    string
	Size  string
	BHash string
}

func GetDocument(id string) (Document, error) {
	doc := Document{}
	idx, err := Client()
	if err != nil {
		return doc, err
	}

	_, err = idx.Search(fmt.Sprintf("_id:%s", id), func(field string, value []byte) bool {
		if field == "filename" {
			doc.Name = string(value)
		}
		if field == "path" {
			doc.Path = string(value)
		}
		if field == "_id" {
			doc.ID = string(value)
		}
		if field == "size" {
			size, err := bluge.DecodeNumericFloat64(value)
			if err != nil {
				logger.Error(err, "error decoding file size")
			}
			doc.Size = fmt.Sprintf("%.0f", size)
		}
		if field == "bhash" {
			doc.BHash = string(value)
		}
		return true
	}, func() bool { return true })

	return doc, err
}

func NeedsIndexing(id string) (bool, error) {
	rs := keyring.New(id)
	idx, err := rindex.NewOffline(config.CurrentIndexPath(), rs.Repository, rs.Password)
	if err != nil {
		return false, err
	}

	missing, err := idx.MissingSnapshots(context.Background())
	if len(missing) > 0 {
		logger.Printf("%d missing snapshots found", len(missing))
	}

	return len(missing) > 0, err
}

func Client() (rindex.Indexer, error) {
	var indexer rindex.Indexer
	if config.PreferredRepo() == "" {
		return indexer, fmt.Errorf("no preferred repository currently set")
	}

	k := keyring.New(config.PreferredRepo())
	//	k.ExportEnv()

	return rindex.NewOffline(config.CurrentIndexPath(), k.Repository, k.Password)
}
