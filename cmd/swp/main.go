package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/blugelabs/bluge"
	"github.com/manifoldco/promptui"
	"github.com/rubiojr/rapi"
	"github.com/rubiojr/rindex"
	"github.com/swampapp/swamp/internal/config"
	"github.com/swampapp/swamp/internal/keyring"
	"github.com/swampapp/swamp/internal/logger"
	"github.com/swampapp/swamp/internal/paths"
	"github.com/swampapp/swamp/internal/queryparser"
	"github.com/urfave/cli/v2"
)

var appCommands []*cli.Command
var globalOptions = rapi.DefaultOptions
var blugeConf bluge.Config
var indexPath string

func main() {
	app := &cli.App{
		Name:     "swp",
		Commands: []*cli.Command{},
		Version:  "v0.1.0",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:     "debug",
				Usage:    "Enable debugging",
				Required: false,
			},
		},
	}

	cmd := &cli.Command{
		Name:   "search",
		Usage:  "Search the index",
		Action: doSearch,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "repo",
				Usage:    "Repository to query",
				Required: false,
			},
		},
	}
	appCommands = append(appCommands, cmd)

	cmd = &cli.Command{
		Name:   "add-repo",
		Usage:  "Add new repository",
		Action: addRepo,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "preferred",
				Usage:    "The preferred repository",
				Required: false,
			},
		},
	}
	appCommands = append(appCommands, cmd)

	app.Commands = append(app.Commands, appCommands...)
	err := app.Run(os.Args)
	if err != nil {
		println(fmt.Sprintf("\n🛑 %s.", err))
		os.Exit(1)
	}
}

func addRepo(c *cli.Context) error {
	prompt := promptui.Prompt{
		Label: "Repository Name",
	}
	name, err := prompt.Run()
	if err != nil {
		if err.Error() == "^C" {
			return fmt.Errorf("aborted")
		}
		return err
	}

	prompts := promptui.Select{
		Label: "Repository Type",
		Items: []string{"Local/Rest", "S3"},
	}
	_, rtype, err := prompts.Run()
	if err != nil {
		if err.Error() == "^C" {
			return fmt.Errorf("aborted")
		}
		return err
	}

	validate := func(input string) error {
		if rtype == "S3" && !strings.HasPrefix(input, "s3:") {
			return errors.New("An S3 repository URI should start with s3:")
		}
		return nil
	}

	prompt = promptui.Prompt{
		Label:    "Repository URI",
		Validate: validate,
	}
	uri, err := prompt.Run()
	if err != nil {
		if err.Error() == "^C" {
			return fmt.Errorf("aborted")
		}
		return err
	}

	prompt = promptui.Prompt{
		Label: "Repository Password",
		Mask:  '*',
	}
	pass, err := prompt.Run()
	if err != nil {
		if err.Error() == "^C" {
			return fmt.Errorf("aborted")
		}
		return err
	}

	var key, secret string
	if rtype == "S3" {
		prompt = promptui.Prompt{
			Label: "ACCESS_ACCESS_KEY",
		}
		key, err = prompt.Run()
		if err != nil {
			if err.Error() == "^C" {
				return fmt.Errorf("aborted")
			}
			return err
		}

		prompt = promptui.Prompt{
			Label: "ACCESS_SECRET_ACCESS_KEY",
			Mask:  '*',
		}
		secret, err = prompt.Run()
		if err != nil {
			if err.Error() == "^C" {
				return fmt.Errorf("aborted")
			}
			return err
		}

		os.Setenv("AWS_ACCESS_KEY", key)
		os.Setenv("AWS_SECRET_ACCESS_KEY", secret)
	}

	fmt.Print("Testing repository access...")
	ropts := rapi.DefaultOptions
	ropts.Repo = uri
	ropts.Password = pass
	repo, err := rapi.OpenRepository(ropts)
	retry := false
	if err != nil {
		fmt.Printf("\nRepository credendials failed.")
		prompt = promptui.Prompt{
			Label:     "Retry?",
			IsConfirm: true,
		}
		_, err := prompt.Run()
		if err != nil {
			fmt.Println("aborting.")
			os.Exit(0)
		}
		retry = true
		addRepo(c)
	}

	if retry {
		return nil
	}

	fmt.Println("✅")

	repoID := repo.Config().ID
	rs := keyring.New(repoID)
	rs.Password = pass
	rs.Repository = uri
	if key != "" {
		rs.Var1 = key
		rs.Var2 = secret
	}
	rs.Save()

	config.AddRepository(name, repoID, false)
	config.Save()
	if c.Bool("preferred") {
		config.SetPreferredRepo(repoID)
	}

	fmt.Println("Added!")
	fmt.Println("Now you should open swamp, select the repository and manually index it.")

	return err
}

func doSearch(c *cli.Context) error {
	if _, err := os.Stat(paths.RepositoriesDir()); os.IsNotExist(err) {
		return fmt.Errorf("swamp CLI doesn't currently support indexing repositories.\nRun the swamp app first.")
	}

	if c.Bool("debug") {
		logger.Init(logger.DebugLevel, "swp")
	} else {
		logger.Init(logger.InfoLevel, "swp")
	}

	q := c.Args().Get(0)
	if q == "" {
		return fmt.Errorf("missing query argument")
	}
	q, err := queryparser.ParseQuery(q)
	if err != nil {
		return err
	}

	verbose := c.Bool("verbose")

	fmt.Printf("Searching...\n\n")

	filterField := func(name string) bool {
		if verbose {
			return false
		}
		switch name {
		case "ext", "blobs", "mtime", "repository_id":
			return true
		default:
			return false
		}
	}

	rs := keyring.New(config.PreferredRepo())

	var indexPath, repoName string
	if repoName = c.String("repo"); repoName != "" {
		rn := config.RepoDirFor(repoName)
		if rn == "" {
			return fmt.Errorf("no repository found with name '%s'", repoName)
		}
		indexPath = filepath.Join(config.RepoDirFor(repoName), "index", "swamp.bluge")
	} else {
		indexPath = filepath.Join(config.PreferredRepoDir(), "index", "swamp.bluge")
	}

	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		return fmt.Errorf("repository '%s' needs to be indexed first. Open swamp to do it", repoName)
	}

	rs.ExportEnv()
	idx, err := rindex.NewOffline(indexPath, globalOptions.Repo, globalOptions.Password)
	if err != nil {
		return err
	}

	count, err := idx.Search(q, func(field string, value []byte) bool {
		if !filterField(field) {
			printMetadata(field, value)
		}
		return true
	},
		func() bool {
			fmt.Println()
			return true
		},
	)

	fmt.Printf("Results: %d\n", count)

	return err
}
