package tags

import (
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/swampapp/swamp/internal/index"
	"github.com/swampapp/swamp/internal/settings"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vmihailenco/msgpack/v5"
)

type Tag struct {
	Name  string
	Color string
}

func dbPath() string {
	return filepath.Join(settings.RepoDir(), "tags.db")
}

func For(fileID string) ([]Tag, error) {
	db, err := leveldb.OpenFile(dbPath(), nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Error().Err(err)
		}
	}()

	tagsblob, err := db.Get([]byte(fileID), nil)
	if err != nil {
		return nil, err
	}

	var tags []Tag
	err = msgpack.Unmarshal(tagsblob, &tags)

	return tags, err
}

func Delete(tag string) {
}

func All() ([]Tag, error) {
	var tags []Tag
	tmap := map[string]Tag{}
	db, err := leveldb.OpenFile(dbPath(), nil)
	if err != nil {
		return tags, err
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Error().Err(err)
		}
	}()

	iter := db.NewIterator(nil, nil)
	for iter.Next() {
		var tags []Tag
		err := msgpack.Unmarshal(iter.Value(), &tags)
		if err != nil {
			return tags, err
		}
		for _, t := range tags {
			tmap[t.Name] = t
		}
	}
	iter.Release()

	for _, v := range tmap {
		tags = append(tags, v)
	}

	return tags, nil
}

func GetDocuments(tag string) ([]index.Document, error) {
	var documents []index.Document
	db, err := leveldb.OpenFile(dbPath(), nil)
	if err != nil {
		return documents, err
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Error().Err(err)
		}
	}()

	iter := db.NewIterator(nil, nil)
	for iter.Next() {
		var tags []Tag
		fileID := string(iter.Key())
		err := msgpack.Unmarshal(iter.Value(), &tags)
		if err != nil {
			return documents, err
		}
		if hasTag(tags, tag) {
			doc, err := index.GetDocument(fileID)
			if err != nil {
				return documents, err
			}
			documents = append(documents, doc)
		}
	}
	iter.Release()

	return documents, nil
}

func hasTag(tags []Tag, name string) bool {
	for _, tag := range tags {
		if tag.Name == name {
			return true
		}
	}

	return false
}

func Save(fileID string, tags []Tag) error {
	db, err := leveldb.OpenFile(dbPath(), nil)
	if err != nil {
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Error().Err(err)
		}
	}()

	mtags, err := msgpack.Marshal(tags)
	if err != nil {
		return err
	}
	return db.Put([]byte(fileID), mtags, nil)
}
