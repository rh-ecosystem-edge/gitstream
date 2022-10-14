package cli

import (
	"errors"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/go-git/go-git/v5"
	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
	"github.com/qbarrand/gitstream/internal/config"
	gh "github.com/qbarrand/gitstream/internal/github"
	"github.com/qbarrand/gitstream/internal/gitstream"
	"github.com/qbarrand/gitstream/internal/gitutils"
	"github.com/qbarrand/gitstream/internal/intents"
	"github.com/qbarrand/gitstream/internal/markup"
	"github.com/urfave/cli/v2"
)

type App struct {
	Config *config.Config
	Logger logr.Logger
}

func (a *App) GetCLIApp() *cli.App {
	const logLevelFlagName = "log-level"

	var configPath string

	commit := getGitCommit()

	app := cli.NewApp()

	app.Action = cli.ShowAppHelp

	app.Authors = []*cli.Author{
		{
			Name:  "Quentin Barrand",
			Email: "quba@redhat.com",
		},
	}

	app.Version = "0.0.1-" + commit

	app.Before = func(c *cli.Context) error {
		cfg, err := config.ReadConfigFile(configPath)
		if err != nil {
			return fmt.Errorf("could not read config: %v", err)
		}

		a.Config = cfg

		logLevel := cfg.LogLevel

		if c.IsSet(logLevelFlagName) {
			logLevel = c.Int(logLevelFlagName)
		}

		stdr.SetVerbosity(logLevel)
		a.Logger.Info("Build information", "commit", commit)
		return nil
	}

	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "config",
			Value:       ".github/gitstream.yml",
			Destination: &configPath,
		},
		&cli.IntFlag{Name: "log-level"},
	}

	app.Usage = "Synchronization tool between an upstream and a downstream repository on GitHub"

	app.Commands = []*cli.Command{
		{
			Name:   "diff",
			Action: a.diff,
			Usage:  "List upstream commits and try to find them downstream",
		},
		{
			Name:   "sync",
			Action: a.sync,
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "dry-run",
					Usage: "if true, no code is pushed and no content is created through the API",
				},
			},
			Usage: "Try to apply missing upstream commits to the downstream repository",
		},
	}

	return app
}

func getGitHubTokenFromEnv() (string, error) {
	token, found := os.LookupEnv("GITHUB_TOKEN")
	if !found {
		return "", errors.New("GITHUB_TOKEN: undefined or empty variable")
	}

	return token, nil
}

func (a *App) diff(c *cli.Context) error {
	ctx := c.Context

	token, err := getGitHubTokenFromEnv()
	if err != nil {
		return fmt.Errorf("could not create a GitHub client: %v", err)
	}

	gc := gh.NewGitHubClient(ctx, token)

	repoName, err := gh.ParseRepoName(a.Config.Downstream.GitHubRepoName)
	if err != nil {
		return fmt.Errorf("%q: invalid repository name", a.Config.Downstream.GitHubRepoName)
	}

	repo, err := git.PlainOpenWithOptions(a.Config.Downstream.LocalRepoPath, &git.PlainOpenOptions{})
	if err != nil {
		return fmt.Errorf("could not open the downstream repo: %v", err)
	}

	finder, err := markup.NewFinder(a.Config.CommitMarkup)
	if err != nil {
		return fmt.Errorf("could not create the markup finder: %v", err)
	}

	d := gitstream.Diff{
		Differ: gitutils.NewDiffer(
			gitutils.NewHelper(repo, a.Logger),
			intents.NewIntentsGetter(finder, gc, a.Logger),
			a.Logger,
		),
		DiffConfig:           a.Config.Diff,
		DownstreamMainBranch: a.Config.Downstream.MainBranch,
		Logger:               a.Logger,
		RepoName:             repoName,
		Repo:                 repo,
		UpstreamConfig:       a.Config.Upstream,
	}

	return d.Run(ctx)
}

func (a *App) sync(c *cli.Context) error {
	ctx := c.Context

	token, err := getGitHubTokenFromEnv()
	if err != nil {
		return fmt.Errorf("could not create a GitHub client: %v", err)
	}

	gc := gh.NewGitHubClient(ctx, token)

	repoName, err := gh.ParseRepoName(a.Config.Downstream.GitHubRepoName)
	if err != nil {
		return fmt.Errorf("%q: invalid repository name", a.Config.Downstream.GitHubRepoName)
	}

	repoPath := a.Config.Downstream.LocalRepoPath

	repo, err := git.PlainOpenWithOptions(repoPath, &git.PlainOpenOptions{})
	if err != nil {
		return fmt.Errorf("could not open the downstream repo: %v", err)
	}

	helper := gitutils.NewHelper(repo, a.Logger)

	finder, err := markup.NewFinder(a.Config.CommitMarkup)
	if err != nil {
		return fmt.Errorf("could not create the markup finder: %v", err)
	}

	s := gitstream.Sync{
		CherryPicker: gitutils.NewCherryPicker(a.Config.CommitMarkup, a.Logger, a.Config.Sync.BeforeCommit...),
		Creator:      gh.NewCreator(gc, a.Config.CommitMarkup, repoName),
		Differ: gitutils.NewDiffer(
			helper,
			intents.NewIntentsGetter(finder, gc, a.Logger),
			a.Logger,
		),
		DiffConfig:       a.Config.Diff,
		DownstreamConfig: a.Config.Downstream,
		DryRun:           c.Bool("dry-run"),
		GitHelper:        helper,
		GitHubToken:      token,
		Logger:           a.Logger,
		Repo:             repo,
		RepoName:         repoName,
		UpstreamConfig:   a.Config.Upstream,
	}

	return s.Run(ctx)
}

func getGitCommit() string {
	bi, ok := debug.ReadBuildInfo()
	if ok {
		for _, s := range bi.Settings {
			if s.Key == "vcs.revision" {
				return s.Value
			}
		}
	}

	return ""
}
