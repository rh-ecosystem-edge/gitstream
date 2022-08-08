package hashget

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-logr/logr"
	"github.com/google/go-github/v47/github"
	"github.com/qbarrand/gitstream/internal"
	"github.com/qbarrand/gitstream/internal/collections"
	gh "github.com/qbarrand/gitstream/internal/github"
	"github.com/qbarrand/gitstream/internal/markup"
)

type HashGetter interface {
	GetHashes(ctx context.Context) (collections.CommitSet, error)
}

type GitHubOpenPRsGetter struct {
	finder markup.Finder
	gc     *github.Client
	logger logr.Logger
	repo   *gh.RepoName
}

func NewGitHubOpenPRsGetter(gc *github.Client, repoName *gh.RepoName, finder markup.Finder, logger logr.Logger) *GitHubOpenPRsGetter {
	return &GitHubOpenPRsGetter{
		finder: finder,
		gc:     gc,
		logger: logger.WithValues("repo", repoName),
		repo:   repoName,
	}
}

func (g *GitHubOpenPRsGetter) GetHashes(ctx context.Context) (collections.CommitSet, error) {
	hashes := collections.NewCommitSet()

	opt := &github.PullRequestListOptions{State: "open"}

	for {
		prs, resp, err := g.gc.PullRequests.List(ctx, g.repo.Owner, g.repo.Repo, opt)
		if err != nil {
			return nil, fmt.Errorf("error while listing PRs: %v", err)
		}

		for _, pr := range prs {
			logger := g.logger.WithValues("id", *pr.ID, "title", pr.Title)

			if pr.Body == nil {
				logger.Info("PR body empty; skipping")
				continue
			}

			shas, err := g.finder.FindSHAs(*pr.Body)
			if err != nil {
				return nil, fmt.Errorf("error while looking for SHAs in %q: %v", *pr.Body, err)
			}

			logger.Info("Adding SHAs", "SHAs", shas)

			hashes.Add(shas...)
		}

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return hashes, nil
}

type GitHubIssuesGetter struct {
	finder markup.Finder
	gc     *github.Client
	logger logr.Logger
	repo   *gh.RepoName
}

func NewGitHubIssuesGetter(gc *github.Client, repoName *gh.RepoName, finder markup.Finder, logger logr.Logger) *GitHubIssuesGetter {
	return &GitHubIssuesGetter{
		finder: finder,
		gc:     gc,
		logger: logger.WithValues("repo", repoName),
		repo:   repoName,
	}
}

func (g *GitHubIssuesGetter) GetHashes(ctx context.Context) (collections.CommitSet, error) {
	hashes := collections.NewCommitSet()

	opt := &github.IssueListByRepoOptions{
		Labels: []string{internal.GitStreamLabel},
	}

	for {
		issues, resp, err := g.gc.Issues.ListByRepo(ctx, g.repo.Owner, g.repo.Repo, opt)
		if err != nil {
			return nil, fmt.Errorf("error while listing issues: %v", err)
		}

		for _, issue := range issues {
			logger := g.logger.WithValues("id", *issue.ID, "title", issue.Title)

			if issue.Body == nil {
				logger.Info("Issue body empty; skipping")
				continue
			}

			shas, err := g.finder.FindSHAs(*issue.Body)
			if err != nil {
				return nil, fmt.Errorf("error while looking for SHAs in %q: %v", *issue.Body, err)
			}

			logger.Info("Adding SHAs", "SHAs", shas)

			hashes.Add(shas...)
		}

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return hashes, nil
}

type GitRepoGetter struct {
	finder markup.Finder
	logger logr.Logger
	repo   *git.Repository
	since  *time.Time
}

func NewGitRepoGetter(repo *git.Repository, since *time.Time, finder markup.Finder, logger logr.Logger) *GitRepoGetter {
	return &GitRepoGetter{
		finder: finder,
		logger: logger,
		repo:   repo,
		since:  since,
	}
}

func (g *GitRepoGetter) GetHashes(ctx context.Context) (collections.CommitSet, error) {
	dsCommitIter, err := g.repo.Log(&git.LogOptions{Since: g.since})
	if err != nil {
		return nil, fmt.Errorf("could not get an iterator on the downstream repo: %v", err)
	}

	hashes := collections.NewCommitSet()

	for {
		commit, err := dsCommitIter.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return nil, fmt.Errorf("error while iterating on commits: %v", err)
		}

		hash := commit.Hash

		g.logger.Info("Processing commit", "commit", hash)

		shas, err := g.finder.FindSHAs(commit.Message)
		if err != nil {
			return nil, fmt.Errorf("error while finding SHAs in commit %s: %v", hash, err)
		}

		hashes.Add(shas...)
	}

	return hashes, nil
}
