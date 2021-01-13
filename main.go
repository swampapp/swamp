package main

import (
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/arl/statsviz"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/swampapp/swamp/internal/resources"
	"github.com/swampapp/swamp/internal/ui/mainwindow"
)

var singleInstance sync.Once

func main() {
	for _, a := range os.Args {
		if a == "-v" || a == "--version" {
			fmt.Println("Swamp v" + APP_VERSION)
			os.Exit(0)
		}
	}

	resources.InitResources()

	statsviz.RegisterDefault()
	go func() {
		log.Print(http.ListenAndServe("localhost:6061", nil))
	}()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Print("starting app")

	gtk.Init(nil)
	app, _ := gtk.ApplicationNew("com.github.rubiojr.swamp", glib.APPLICATION_FLAGS_NONE)

	app.Connect("activate", func() {
		singleInstance.Do(func() { activate(app) })
	})

	if exitCode := app.Run(os.Args); exitCode > 0 {
		os.Exit(exitCode)
	}
}

func activate(app *gtk.Application) {
	w, err := mainwindow.New(app)
	if err != nil {
		log.Printf("Failed to create main window: %+v", err)
		panic(err)
	}

	app.AddWindow(w)

	w.Connect("destroy", func() {
	})
}
