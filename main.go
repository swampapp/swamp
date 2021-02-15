package main

import (
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/arl/statsviz"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/swampapp/swamp/internal/config"
	"github.com/swampapp/swamp/internal/logger"
	"github.com/swampapp/swamp/internal/resources"
	"github.com/swampapp/swamp/internal/ui/mainwindow"
)

var singleInstance sync.Once

func main() {
	logger.Init(logger.DebugLevel, "swamp")

	for _, a := range os.Args {
		if a == "-v" || a == "--version" {
			fmt.Printf("Swamp v%s (%s)\n", APP_VERSION, GIT_SHA)
			os.Exit(0)
		}
	}
	_, err := config.Init()
	if err != nil {
		panic(fmt.Errorf("error initializing configuration: %w", err))
	}

	resources.InitResources()

	statsviz.RegisterDefault()
	go func() {
		logger.Print(http.ListenAndServe("localhost:6061", nil))
	}()

	logger.Print("starting app")

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
		logger.Printf("Failed to create main window: %+v", err)
		panic(err)
	}

	app.AddWindow(w)

	w.Connect("destroy", func() {
	})
}
