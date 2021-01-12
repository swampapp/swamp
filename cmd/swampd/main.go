package main

import (
	"fmt"
	"os"

	"github.com/blugelabs/bluge"
	gap "github.com/muesli/go-app-paths"
	"github.com/rubiojr/rapi"
	"github.com/urfave/cli/v2"
)

var appCommands []*cli.Command
var globalOptions = rapi.DefaultOptions
var blugeConf bluge.Config
var indexPath string

func initApp() {
	os.MkdirAll(indexPath, 0755)
	blugeConf = bluge.DefaultConfig(indexPath)
}

func defaultDataDir() string {
	scope := gap.NewScope(gap.User, "swamp")
	dirs, err := scope.DataDirs()
	if err != nil {
		panic(err)
	}
	return dirs[0]
}

func main() {
	var err error
	app := &cli.App{
		Name:     "swampd",
		Commands: []*cli.Command{},
		Version:  "v0.1.0",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "repo",
				Aliases:     []string{"r"},
				EnvVars:     []string{"RESTIC_REPOSITORY"},
				Usage:       "Repository path",
				Required:    false,
				Destination: &globalOptions.Repo,
			},
			&cli.StringFlag{
				Name:        "password",
				Aliases:     []string{"p"},
				EnvVars:     []string{"RESTIC_PASSWORD"},
				Usage:       "Repository password",
				Required:    false,
				Destination: &globalOptions.Password,
				DefaultText: " ",
			},
			&cli.StringFlag{
				Name:        "index-path",
				Usage:       "Index path",
				Required:    true,
				Destination: &indexPath,
			},
			&cli.BoolFlag{
				Name:     "debug",
				Aliases:  []string{"d"},
				Usage:    "Enable debugging",
				Required: false,
			},
		},
	}

	app.Commands = append(app.Commands, appCommands...)
	err = app.Run(os.Args)
	if err != nil {
		println(fmt.Sprintf("\n🛑 %s", err))
		os.Exit(1)
	}
}
