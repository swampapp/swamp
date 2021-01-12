package main

import (
	"fmt"
	"strings"

	"github.com/blugelabs/bluge"
	"github.com/dustin/go-humanize"
	"github.com/muesli/reflow/padding"
	"github.com/muesli/termenv"
)

const (
	headerColor = "#ffb236"
	colPadding  = 20
)

func colorize(str, color string) string {
	out := termenv.String(str)
	p := termenv.ColorProfile()
	return out.Foreground(p.Color(color)).String()
}

func printRow(header, value, color string) {
	fmt.Printf("%s %s\n", padding.String(colorize(header+":", color), colPadding), value)
}

func printMetadata(field string, value []byte) {
	var f string
	if field == "_id" {
		f = "ID"
	} else if field == "repository_id" {
		f = "Repository ID"
	} else {
		f = strings.Title(strings.ReplaceAll(field, "_", " "))
	}

	v := ""
	switch field {
	case "mtime":
		t, err := bluge.DecodeDateTime(value)
		if err != nil {
			v = "error"
		} else {
			v = t.Format("2006-1-2")
		}
	case "updated":
		t, err := bluge.DecodeDateTime(value)
		if err != nil {
			v = "error"
		} else {
			v = t.Format("2006-1-2")
		}
	case "size":
		t, err := bluge.DecodeNumericFloat64(value)
		if err != nil {
			v = "error"
		} else {
			v = humanize.Bytes(uint64(t))
		}
	default:
		v = string(value)
	}

	if v != "" {
		printRow(f, v, headerColor)
	}
}
