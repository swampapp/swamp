package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	_ "net/http/pprof"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/swampapp/swamp/internal/config"
	"github.com/swampapp/swamp/internal/credentials"
	"github.com/swampapp/swamp/internal/logger"
	"github.com/swampapp/swamp/internal/paths"
	"github.com/swampapp/swamp/internal/resources"
	"github.com/swampapp/swamp/internal/ui/assistant"
	"github.com/swampapp/swamp/internal/ui/mainwindow"
)

func main() {
	logger.Init(logger.DebugLevel, "swamp")

	for _, a := range os.Args {
		if a == "-v" || a == "--version" {
			fmt.Printf("Swamp v%s (%s)\n", APP_VERSION, GIT_SHA)
			os.Exit(0)
		}
	}

	go func() {
		log.Println(http.ListenAndServe("localhost:6061", nil))
	}()

	err := paths.Initialize()
	if err != nil {
		panic(fmt.Errorf("error initializing paths: %w", err))
	}

	_, err = config.Init()
	if err != nil {
		panic(fmt.Errorf("error initializing configuration: %w", err))
	}

	resources.InitResources()

	logger.Print("starting app")

	gtk.Init(nil)
	app, _ := gtk.ApplicationNew("com.github.rubiojr.swamp", glib.APPLICATION_FLAGS_NONE)

	screen, _ := gdk.ScreenGetDefault()
	gtk.AddProviderForScreen(screen, resources.CSS(), gtk.STYLE_PROVIDER_PRIORITY_USER)

	app.Connect("activate", func() {
		if credentials.FirstBoot() {
			a := assistant.New()
			app.AddWindow(a)
			a.ShowAll()
			a.WhenDone(func() {
				activate(app)
			})
		} else {
			activate(app)
		}
	})

	if exitCode := app.Run(os.Args); exitCode > 0 {
		os.Exit(exitCode)
	}
}

func activate(app *gtk.Application) {
	w, _ := mainwindow.New(app)
	w.ShowAll()
	app.AddWindow(w)
	w.Connect("destroy", func() {
	})
}
