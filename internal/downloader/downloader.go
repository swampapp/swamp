package downloader

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/Jeffail/tunny"

	"github.com/swampapp/swamp/internal/index"
	"github.com/swampapp/swamp/internal/logger"
	"github.com/swampapp/swamp/internal/settings"
	"github.com/syndtr/goleveldb/leveldb"
)

type EventType uint8

type Document struct {
	index.Document
	DateTime time.Time
}

type Observer interface {
	NotifyCallback(event DownloadEvent)
	Name() string
}

const (
	EventStart EventType = iota
	EventStop
	EventFinished
	EventQueueEmpty
	EventError
)

var once sync.Once
var instance *Downloader

const maxWorkers = 5

var m = &sync.Mutex{}

type Downloader struct {
	observers  sync.Map
	pool       *tunny.Pool
	inProgress []Document
}

type DownloadEvent struct {
	Type   EventType
	FileID string
}

type downloadRequest struct {
	fileID    string
	open      bool
	exportDir string
	name      string
}

var dcache *leveldb.DB

func init() {
	if err := os.MkdirAll(settings.DownloadsDir(), 0755); err != nil {
		logger.Error(err, "")
	}
}

func Instance() *Downloader {
	once.Do(func() {
		var err error
		dcache, err = leveldb.OpenFile(filepath.Join(settings.DownloadsDir(), "index"), nil)
		if err != nil {
			panic(err)
		}
		pool := tunny.NewFunc(maxWorkers, func(i interface{}) interface{} {
			req := i.(downloadRequest)
			err := instance.downloadFileID(req.fileID)
			if err != nil {
				logger.Error(err, "")
				return err
			}

			if req.exportDir != "" && req.name != "" {
				err := Export(req.fileID, req.name, req.exportDir)
				if err != nil {
					logger.Errorf(err, "error exporting file '%s'", req.name)
					return err
				}
			}

			if req.open {
				if err := Open(req.fileID); err != nil {
					logger.Error(err, "")
				}
			}

			return err
		})
		instance = &Downloader{pool: pool, inProgress: []Document{}, observers: sync.Map{}}
	})

	return instance
}

func (d *Downloader) Downloaded() ([]Document, error) {
	docs := []Document{}
	iter := dcache.NewIterator(nil, nil)
	for iter.Next() {
		var doc Document
		doc.ID = string(iter.Key())
		dec := gob.NewDecoder(bytes.NewReader(iter.Value()))
		err := dec.Decode(&doc)
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	iter.Release()

	sort.Slice(docs, func(i, j int) bool {
		return docs[i].DateTime.After(docs[j].DateTime)
	})
	return docs, iter.Error()
}

func (d *Downloader) IsDownloading() bool {
	return d.pool.QueueLength() > 0
}

func (d *Downloader) InProgress() int {
	return len(d.inProgress)
}

func (d *Downloader) DownloadsInProgress() []Document {
	return d.inProgress
}

func (d *Downloader) IsInProgress(id string) bool {
	for _, f := range d.inProgress {
		if f.ID == id {
			return true
		}
	}
	return false
}

func (d *Downloader) Remove(fileID string) error {
	err := dcache.Delete([]byte(fileID), nil)
	if err != nil {
		m.Unlock()
		return err
	}

	return os.Remove(PathFromID(fileID))
}

func (d *Downloader) WasDownloaded(fileID string) (bool, error) {
	var err error
	_, err = dcache.Get([]byte(fileID), nil)
	if err == nil {
		return true, nil
	}

	if err == leveldb.ErrNotFound {
		return false, nil
	}

	return false, err
}

func (d *Downloader) addInProgress(doc Document) {
	m.Lock()
	defer m.Unlock()
	d.inProgress = append(d.inProgress, doc)
}

func (d *Downloader) Download(fileID string) {
	go d.pool.Process(downloadRequest{fileID: fileID, open: false})
}

func (d *Downloader) DownloadAndOpen(fileID string) {
	go d.pool.Process(downloadRequest{fileID: fileID, open: true})
}

func (d *Downloader) DownloadAndExport(fileID, name, targetDir string) {
	go d.pool.Process(downloadRequest{fileID: fileID, name: name, exportDir: targetDir})
}

func (d *Downloader) downloadFileID(fileID string) error {
	doc, err := index.GetDocument(fileID)
	if err != nil {
		logger.Errorf(err, "file %s not found in index", fileID)
		return err
	}
	dpath := PathFromID(fileID)
	if _, err := os.Stat(dpath); err == nil {
		logger.Print("already downloaded ", fileID)
		return fmt.Errorf("file %s already downloaded", fileID)
	}

	ddoc := Document{}
	ddoc.DateTime = time.Now()
	ddoc.Document = doc

	d.addInProgress(ddoc)
	d.NotifyObservers(DownloadEvent{Type: EventStart, FileID: fileID})

	idx, err := index.Client()
	if err != nil {
		d.NotifyObservers(DownloadEvent{Type: EventError, FileID: fileID})
		logger.Error(err, "error initializing the index")
		return err
	}

	err = os.MkdirAll(filepath.Dir(dpath), 0755)
	if err != nil {
		d.NotifyObservers(DownloadEvent{Type: EventError, FileID: fileID})
		return err
	}

	dest, err := os.Create(dpath + ".tmp")
	if err != nil {
		d.NotifyObservers(DownloadEvent{Type: EventError, FileID: fileID})
		logger.Error(err, "error creating download tmp file")
		return err
	}
	defer func() {
		if err := dest.Close(); err != nil {
			logger.Error(err, "")
		}
	}()

	logger.Print("downloading ", fileID)
	err = idx.Fetch(context.Background(), fileID, dest)
	if err != nil {
		d.NotifyObservers(DownloadEvent{Type: EventError, FileID: fileID})
		logger.Error(err, "error downloading file")
		return err
	}

	if err := os.Rename(dest.Name(), dpath); err != nil {
		logger.Errorf(err, "error moving file %s to %s", dest.Name(), dpath)
		return err
	}

	// doc downloaded successfully, add it to the cache
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err = enc.Encode(ddoc)
	if err != nil {
		return err
	}

	err = dcache.Put([]byte(fileID), buf.Bytes(), nil)
	if err != nil {
		logger.Error(err, "error adding file to leveldb")
		return err
	}

	if !d.removeInProgress(fileID) {
		logger.Debug("failed to remove from in progress")
	}

	d.NotifyObservers(DownloadEvent{Type: EventFinished, FileID: fileID})
	logger.Print("downloaded ", fileID)

	return err
}

func (d *Downloader) removeInProgress(fid string) bool {
	m.Lock()
	defer m.Unlock()

	for i, item := range d.inProgress {
		if item.ID == fid {
			d.inProgress = append(d.inProgress[:i], d.inProgress[i+1:]...)
			if len(d.inProgress) == 0 {
				d.NotifyObservers(DownloadEvent{Type: EventQueueEmpty, FileID: fid})
			}
			return true
		}
	}

	return false
}

// PathFromID returns the full path to a downloaded file
func PathFromID(fileID string) string {
	return filepath.Join(settings.DownloadsDir(), fileID[:2], fileID)
}

func Open(fid string) error {
	fpath := PathFromID(fid)
	logger.Print("Opening ", fpath)
	cmd := exec.Command("/usr/bin/xdg-open", fpath)
	err := cmd.Run()
	if err != nil {
		logger.Print("error opening ", fpath)
	}
	return err
}

func safeExportName(dest string) string {
	count := 0
	suffix := ""
	ext := filepath.Ext(dest)
	name := dest[0 : len(dest)-len(ext)]

	var findSafeName func(string) string
	findSafeName = func(d string) string {
		_, err := os.Stat(d)
		if err != nil {
			newn := fmt.Sprintf("%s%s%s", name, suffix, ext)
			return newn
		}

		count++
		suffix = fmt.Sprintf("_%d", count)
		return findSafeName(fmt.Sprintf("%s%s%s", name, suffix, ext))
	}

	return findSafeName(dest)
}

func Export(fid, name, target string) error {
	fpath := PathFromID(fid)
	logger.Printf("Exporting %s to %s", fpath, target)
	fi, err := os.Stat(fpath)
	if err != nil {
		return err
	}

	if !fi.Mode().IsRegular() {
		return fmt.Errorf("can't copy non-regular source file %s (%q)", fi.Name(), fi.Mode().String())
	}

	dfi, err := os.Stat(target)
	if err != nil {
		return err
	}

	if !(dfi.Mode().IsDir()) {
		return fmt.Errorf("target '%s' should be a valid directory", target)
	}

	in, err := os.Open(fpath)
	if err != nil {
		return err
	}
	defer func() {
		if err := in.Close(); err != nil {
			logger.Error(err, "")
		}
	}()

	sn := safeExportName(filepath.Join(target, name))
	out, err := os.Create(sn)
	if err != nil {
		return err
	}
	defer func() {
		if err := out.Close(); err != nil {
			logger.Error(err, "")
		}
	}()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	logger.Printf("exported file %s as %s", fid, sn)

	return err
}

func (d *Downloader) AddObserver(observer Observer) {
	d.observers.Store(observer, struct{}{})
}

func (d *Downloader) RemoveObserver(observer Observer) {
	d.observers.Delete(observer)
}

func (d *Downloader) NotifyObservers(event DownloadEvent) {
	d.observers.Range(func(key, value interface{}) bool {
		if key == nil {
			return false
		}

		observer := key.(Observer)
		observer.NotifyCallback(event)
		return true
	})
}
