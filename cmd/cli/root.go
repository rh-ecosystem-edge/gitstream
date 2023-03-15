package cli

import (
	"errors"
	"fmt"
	"os"
	"runtime/debug"

	ghcli "github.com/cli/go-gh"
	"github.com/cli/go-gh/pkg/api"
	"github.com/go-git/go-git/v5"
	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
	"github.com/qbarrand/gitstream/internal/config"
	gh "github.com/qbarrand/gitstream/internal/github"
	"github.com/qbarrand/gitstream/internal/gitstream"
	"github.com/qbarrand/gitstream/internal/gitutils"
	"github.com/qbarrand/gitstream/internal/intents"
	"github.com/qbarrand/gitstream/internal/markup"
	"github.com/qbarrand/gitstream/internal/owners"
	"github.com/urfave/cli/v2"
)

type App struct {
	Config *config.Config
	Logger logr.Logger
}

func (a *App) GetCLIApp() *cli.App {
	const logLevelFlagName = "log-level"

	var (
		configPath string
		flagDryRun = &cli.BoolFlag{
			Name:  "dry-run",
			Usage: "if true, no code is pushed and no content is created through the API",
		}
	)

	commit := getGitCommit()

	app := cli.NewApp()

	app.Action = cli.ShowAppHelp

	app.Authors = []*cli.Author{
		{
			Name:  "Quentin Barrand",
			Email: "quba@redhat.com",
		},
		{
			Name:  "Yoni Bettan",
			Email: "yonibettan@gmail.com",
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
			Name:   "delete-remote-branches",
			Action: a.deleteRemoteBranches,
			Usage:  "Delete all branches with the GitStream prefix on the downstream repository",
		},
		{
			Name:   "diff",
			Action: a.diff,
			Usage:  "List upstream commits and try to find them downstream",
		},
		{
			Name:   "make-oldest-draft-pr-ready",
			Action: a.makeOldestDraftPRReady,
			Flags:  []cli.Flag{flagDryRun},
			Usage:  "Make the oldest draft GitStream PR ready",
		},
		{
			Name:   "sync",
			Action: a.sync,
			Flags:  []cli.Flag{flagDryRun},
			Usage:  "Try to apply missing upstream commits to the downstream repository",
		},
		{
			Name:   "assign",
			Action: a.assign,
			Flags:  []cli.Flag{flagDryRun},
			Usage:  "Assign open issues to the original commit author",
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

func (a *App) deleteRemoteBranches(c *cli.Context) error {
	ctx := c.Context

	token, err := getGitHubTokenFromEnv()
	if err != nil {
		return fmt.Errorf("could not create a GitHub client: %v", err)
	}

	repo, err := git.PlainOpenWithOptions(a.Config.Downstream.LocalRepoPath, &git.PlainOpenOptions{})
	if err != nil {
		return fmt.Errorf("could not open the downstream repo: %v", err)
	}

	d := gitstream.DeleteRemoteBranches{
		GitHubToken: token,
		Logger:      a.Logger,
		Repo:        repo,
	}

	return d.Run(ctx)
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

func (a *App) makeOldestDraftPRReady(c *cli.Context) error {
	ctx := c.Context

	token, err := getGitHubTokenFromEnv()
	if err != nil {
		return fmt.Errorf("could not create a GitHub client: %v", err)
	}

	gc := gh.NewGitHubClient(ctx, token)

	ghgql, err := ghcli.GQLClient(&api.ClientOptions{AuthToken: token})
	if err != nil {
		return fmt.Errorf("could not create a new GraphQL client: %v", err)
	}

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

	u := gitstream.Undraft{
		DryRun:         c.Bool("dry-run"),
		Finder:         finder,
		GitHelper:      gitutils.NewHelper(repo, a.Logger),
		Logger:         a.Logger,
		PRHelper:       gh.NewPRHelper(gc, ghgql, a.Config.CommitMarkup, repoName),
		Repo:           repo,
		RepoName:       repoName,
		UpstreamConfig: a.Config.Upstream,
	}

	return u.Run(ctx)
}

func (a *App) sync(c *cli.Context) error {
	ctx := c.Context

	token, err := getGitHubTokenFromEnv()
	if err != nil {
		return fmt.Errorf("could not create a GitHub client: %v", err)
	}

	gc := gh.NewGitHubClient(ctx, token)

	ghgql, err := ghcli.GQLClient(&api.ClientOptions{AuthToken: token})
	if err != nil {
		return fmt.Errorf("could not create a new GraphQL client: %v", err)
	}

	repoName, err := gh.ParseRepoName(a.Config.Downstream.GitHubRepoName)
	if err != nil {
		return fmt.Errorf("%q: invalid repository name", a.Config.Downstream.GitHubRepoName)
	}

	repo, err := git.PlainOpenWithOptions(a.Config.Downstream.LocalRepoPath, &git.PlainOpenOptions{})
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
		IssueHelper:      gh.NewIssueHelper(gc, a.Config.CommitMarkup, repoName),
		Logger:           a.Logger,
		PRHelper:         gh.NewPRHelper(gc, ghgql, a.Config.CommitMarkup, repoName),
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

func (a *App) assign(c *cli.Context) error {
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

	upstreamRepoName, err := gh.ParseURL(a.Config.Upstream.URL)
	if err != nil {
		return fmt.Errorf("%q: invalid URL", a.Config.Upstream)
	}

	repo, err := git.PlainOpenWithOptions(a.Config.Downstream.LocalRepoPath, &git.PlainOpenOptions{})
	if err != nil {
		return fmt.Errorf("could not open the downstream repo: %v", err)
	}

	finder, err := markup.NewFinder(a.Config.CommitMarkup)
	if err != nil {
		return fmt.Errorf("could not create the markup finder: %v", err)
	}

	u := gitstream.Assign{
		GC:               gc,
		DryRun:           c.Bool("dry-run"),
		Finder:           finder,
		GitHelper:        gitutils.NewHelper(repo, a.Logger),
		Logger:           a.Logger,
		IssueHelper:      gh.NewIssueHelper(gc, a.Config.CommitMarkup, repoName),
		UserHelper:       gh.NewUserHelper(gc, repoName),
		Repo:             repo,
		RepoName:         upstreamRepoName,
		UpstreamConfig:   a.Config.Upstream,
		DownstreamConfig: a.Config.Downstream,
		OwnersHelper:     owners.NewOwnersHelper(),
	}

	return u.Run(ctx)
}
