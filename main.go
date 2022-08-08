package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-logr/stdr"
	"github.com/qbarrand/gitstream/cmd/cli"
	"github.com/qbarrand/gitstream/internal/collections"
	gh "github.com/qbarrand/gitstream/internal/github"
	"github.com/qbarrand/gitstream/internal/gitutils"
	"github.com/qbarrand/gitstream/internal/hashget"
	"github.com/qbarrand/gitstream/internal/markup"
)

const (
	appName            = "gitstream"
	gitStreamPrefix    = "gs-"
	upstreamRemoteName = gitStreamPrefix + "upstream"
)

func logProc(name string, rc io.ReadCloser) {
	s := bufio.NewScanner(rc)

	for s.Scan() {
		log.Printf("[%s] %s", name, s.Text())
	}

	if err := s.Err(); err != nil {
		log.Fatalf("Scanner error for %s: %v", name, err)
	}
}

func main() {
	logger := stdr.New(
		log.New(os.Stdout, "", log.Lshortfile),
	)

	cl, err := cli.Parse(os.Args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}

		logger.Error(err, "Could not parse command-line arguments")
		os.Exit(1)
	}

	finder, err := markup.NewFinder("Upstream-Commit")
	if err != nil {
		logger.Error(err, "Could not create a new markup finder")
		os.Exit(1)
	}

	githubToken := os.Getenv("GITHUB_TOKEN")

	if githubToken == "" {
		logger.Info("GITHUB_TOKEN empty or not set")
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	logger.Info("Creating a new GitHub client")

	gc := gh.NewGitHubClient(ctx, githubToken)

	issueCreator := gh.NewIssueCreator(gc)

	logger.Info("Opening downstream repo", "path", cl.DownstreamRepoPath)

	repo, err := git.PlainOpenWithOptions(cl.DownstreamRepoPath, &git.PlainOpenOptions{})
	if err != nil {
		log.Fatalf("Could not open the upstream repo: %v", err)
	}

	dsCommitsHashGetter := hashget.NewGitRepoGetter(repo, cl.DownstreamSince.Time(), finder, logger)

	repoName, err := gh.ParseRepoName(cl.DownstreamRepoName)
	if err != nil {
		logger.Error(err, "Could not parse the repo name on GitHub")
		os.Exit(1)
	}

	dsIssueHashGetter := hashget.NewGitHubIssuesGetter(gc, repoName, finder, logger)
	dsPRHashGetter := hashget.NewGitHubOpenPRsGetter(gc, repoName, finder, logger)

	commitHashes, err := dsCommitsHashGetter.GetHashes(ctx)
	if err != nil {
		logger.Error(err, "Could not get hashes from commits")
		os.Exit(1)
	}

	issueHashes, err := dsIssueHashGetter.GetHashes(ctx)
	if err != nil {
		logger.Error(err, "Could not get hashes from issues")
		os.Exit(1)
	}

	prHashes, err := dsPRHashGetter.GetHashes(ctx)
	if err != nil {
		logger.Error(err, "Could not get hashes from PRs")
		os.Exit(1)
	}

	allDownstreamHashes := collections.NewCommitSet()
	allDownstreamHashes.Merge(commitHashes, issueHashes, prHashes)

	upstreamRemoteConfig := config.RemoteConfig{
		Name: upstreamRemoteName,
		URLs: []string{cl.UpstreamURL},
	}

	gitHelper := gitutils.NewHelper(logger)

	logger.Info("(Re)creating remote", "name", upstreamRemoteName, "url", cl.UpstreamURL)

	remote, err := gitHelper.RecreateRemote(repo, &upstreamRemoteConfig)
	if err != nil {
		logger.Error(err, "Error adding remote", "name", upstreamRemoteName)
		os.Exit(1)
	}

	if err := remote.FetchContext(ctx, &git.FetchOptions{RemoteName: upstreamRemoteName}); err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		logger.Error(err, "Could not fetch remote", "name", upstreamRemoteName)
		os.Exit(1)
	}

	dsSince := cl.DownstreamSince.Time()

	logger.Info("Looking at downstream commits", "since", dsSince)

	from := plumbing.NewRemoteReferenceName(upstreamRemoteName, "main")

	ref, err := repo.Reference(from, true)
	if err != nil {
		logger.Error(err, "Could not get the reference", "remote", upstreamRemoteName, "branch", "main")
		os.Exit(1)
	}

	lo := git.LogOptions{
		From:  ref.Hash(),
		Since: cl.UpstreamSince.Time(),
	}

	usCommitIter, err := repo.Log(&lo)
	if err != nil {
		logger.Error(err, "Could not get a commit iterator for downstream")
		os.Exit(1)
	}

	upstreamRepo := gh.Repository{
		Name: *repoName,
		URL:  cl.UpstreamURL,
	}

	for {
		commit, err := usCommitIter.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			logger.Error(err, "Error while iterating on commits")
			os.Exit(1)
		}

		hash := commit.Hash

		logger := logger.WithValues("SHA", hash, "message", commit.Message)

		logger.Info("Processing upstream commit")

		if !allDownstreamHashes.Contains(hash) {
			logger.Info("Commit found in the downstream repository; skipping")
			continue
		}

		logger.Info("Cherry-picking the commit")

		sha := hash.String()

		branchName := gitStreamPrefix + sha

		bc := config.Branch{
			Name:   branchName,
			Remote: "origin",
		}

		logger = logger.WithValues("branch name", branchName)

		logger.Info("(Re)creating branch")

		if err := gitHelper.RecreateBranch(repo, &bc); err != nil {
			logger.Error(err, "Error while (re)creating branch")
			os.Exit(1)
		}

		args := []string{"cherry-pick", sha}

		cmd := exec.CommandContext(ctx, "git", args...)
		cmd.Dir = cl.DownstreamRepoPath

		logger.Info("Running git", "args", args, "directory", cl.DownstreamRepoPath)

		out, err := cmd.CombinedOutput()
		if err != nil {
			fields := []string{"SHA", sha}

			exitErr := &exec.ExitError{}
			var exitCode *int

			if errors.As(err, &exitErr) {
				code := exitErr.ExitCode()
				exitCode = &code

				fields = append(fields, "exit code", strconv.Itoa(code))
			}

			logger.Error(err, "Error while cherry-picking; creating issue", "SHA", sha)

			pe := gh.ProcessError{
				Output:     string(out),
				ReturnCode: exitCode,
			}

			issue, err := issueCreator.CreateIssue(ctx, &pe, &upstreamRepo, commit)
			if err != nil {
				logger.Error(err, "Could not create issue")
				os.Exit(1)
			}

			logger.Info("Successfully created issue", "URL", issue.URL)

			continue
		}

		logger.Info("Pushing")

		if err := repo.PushContext(ctx, &git.PushOptions{}); err != nil {
			logger.Error(err, "Error while pushing")
			os.Exit(1)
		}
	}
}
