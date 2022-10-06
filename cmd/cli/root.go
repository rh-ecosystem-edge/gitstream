package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
	gh "github.com/qbarrand/gitstream/internal/github"
	"github.com/qbarrand/gitstream/internal/gitstream"
	"github.com/qbarrand/gitstream/internal/gitutils"
	"github.com/qbarrand/gitstream/internal/intents"
	"github.com/qbarrand/gitstream/internal/markup"
	"github.com/urfave/cli/v2"
)

type App struct {
	Finder        markup.Finder
	IntentsGetter intents.Getter
	Logger        logr.Logger
}

func (a *App) GetCLIApp() *cli.App {
	var (
		flagSince = &cli.TimestampFlag{
			Name:   "since",
			Layout: "2006-02-01",
			Usage:  "only look at upstream commits on or after that date (YYYY-MM-DD)",
		}
		flagDsRepoName = &cli.StringFlag{
			Name:  "downstream-repo-name",
			Usage: "owner/repo",
		}
		flagDsRepoPath = &cli.StringFlag{
			Name:  "downstream-repo-path",
			Value: ".",
			Usage: "path to the local copy of the downstream repo",
		}
		flagUpstreamRef = &cli.StringFlag{
			Name:  "upstream-ref",
			Value: "main",
			Usage: "name of the upstream branch we want to read commits from",
		}
		flagUpstreamURL = &cli.StringFlag{
			Name:     "upstream-url",
			Required: true,
			Usage:    "Git URL of the upstream repository",
		}
		logLevel int
	)

	app := cli.NewApp()

	app.Action = cli.ShowAppHelp

	app.Before = func(context *cli.Context) error {
		stdr.SetVerbosity(logLevel)
		return nil
	}

	app.Flags = []cli.Flag{
		&cli.IntFlag{
			Name:        "log-level",
			Destination: &logLevel,
		},
	}

	app.Usage = "Synchronization tool between an upstream and a downstream repository on GitHub"

	app.Commands = []*cli.Command{
		{
			Name:   "diff",
			Action: a.diff,
			Flags: []cli.Flag{
				flagSince,
				flagDsRepoName,
				flagDsRepoPath,
				flagUpstreamRef,
				flagUpstreamURL,
			},
			Usage: "List upstream commits and try to find them downstream",
		},
		{
			Name:   "sync",
			Action: a.sync,
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "dry-run",
					Usage: "if true, no code is pushed and no content is created through the API",
				},
				flagSince,
				flagDsRepoName,
				flagDsRepoPath,
				flagUpstreamRef,
				flagUpstreamURL,
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

	rawRepoName := c.String("downstream-repo-name")
	repoName, err := gh.ParseRepoName(rawRepoName)
	if err != nil {
		return fmt.Errorf("%q: invalid repository name", rawRepoName)
	}

	repo, err := git.PlainOpenWithOptions(c.String("downstream-repo-path"), &git.PlainOpenOptions{})
	if err != nil {
		return fmt.Errorf("could not open the downstream repo: %v", err)
	}

	d := gitstream.Diff{
		Differ:                  gitutils.NewDiffer(a.Logger),
		DownstreamIntentsGetter: gitutils.NewDownstreamIntentsGetter(a.Logger, a.IntentsGetter, gc),
		Repo:                    repo,
		DownstreamRepoName:      repoName,
		GitHelper:               gitutils.NewHelper(a.Logger),
		Logger:                  a.Logger,
		UpstreamRef:             c.String("upstream-ref"),
		UpstreamSince:           c.Timestamp("since"),
		UpstreamURL:             c.String("upstream-url"),
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

	rawRepoName := c.String("downstream-repo-name")
	repoName, err := gh.ParseRepoName(rawRepoName)
	if err != nil {
		return fmt.Errorf("%q: invalid repository name", rawRepoName)
	}

	repoPath := c.String("downstream-repo-path")

	repo, err := git.PlainOpenWithOptions(repoPath, &git.PlainOpenOptions{})
	if err != nil {
		return fmt.Errorf("could not open the downstream repo: %v", err)
	}

	s := gitstream.Sync{
		Creator:                 gh.NewCreator(gc),
		Differ:                  gitutils.NewDiffer(a.Logger),
		DryRun:                  c.Bool("dry-run"),
		DownstreamIntentsGetter: gitutils.NewDownstreamIntentsGetter(a.Logger, a.IntentsGetter, gc),
		DownstreamRepoName:      repoName,
		GitHelper:               gitutils.NewHelper(a.Logger),
		Logger:                  a.Logger,
		Repo:                    gitutils.NewRepoWrapper(repo, a.Logger, repoPath, token),
		UpstreamRef:             c.String("upstream-ref"),
		UpstreamSince:           c.Timestamp("since"),
		UpstreamURL:             c.String("upstream-url"),
	}

	return s.Run(ctx)
}
