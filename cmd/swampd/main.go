package main

import (
	"fmt"
	"os"

	"github.com/rubiojr/rapi"
	"github.com/swampapp/swamp/internal/logger"
	"github.com/swampapp/swamp/internal/version"
	"github.com/urfave/cli/v2"
)

var appCommands []*cli.Command
var globalOptions = rapi.DefaultOptions
var indexPath string

func main() {
	var err error
	app := &cli.App{
		Name:     "swampd",
		Commands: []*cli.Command{},
		Version:  fmt.Sprintf("swampd v%s (%s)\n", version.APP_VERSION, version.GIT_SHA),
		Before: func(c *cli.Context) error {
			if c.Bool("debug") {
				logger.Init(logger.DebugLevel, "swampd")
			} else {
				logger.Init(logger.InfoLevel, "swampd")
			}
			return nil
		},
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
