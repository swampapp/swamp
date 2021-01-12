package main

import (
	"path/filepath"
	"time"

	"github.com/blugelabs/bluge"
	"github.com/rubiojr/rapi/repository"
	"github.com/rubiojr/rapi/restic"
)

type FileDocumentBuilder struct{}

func (i FileDocumentBuilder) BuildDocument(fileID string, node *restic.Node, repo *repository.Repository) *bluge.Document {
	return bluge.NewDocument(fileID).
		AddField(bluge.NewTextField("ext", filepath.Ext(node.Name)).StoreValue()).
		AddField(bluge.NewDateTimeField("updated", time.Now()).StoreValue())
}
