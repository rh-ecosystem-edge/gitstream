package intents

import (
	"context"
	"fmt"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-logr/logr"
	"github.com/google/go-github/v47/github"
	"github.com/qbarrand/gitstream/internal"
	gh "github.com/qbarrand/gitstream/internal/github"
	"github.com/qbarrand/gitstream/internal/markup"
)

type CommitIntents map[plumbing.Hash]string

func MergeCommitIntents(cis ...CommitIntents) CommitIntents {
	length := 0

	for _, ci := range cis {
		length += len(ci)
	}

	res := make(CommitIntents, length)

	for _, ci := range cis {
		for k, v := range ci {
			res[k] = v
		}
	}

	return res
}

//go:generate mockgen -source=getter.go -package=intents -destination=mock_getter.go

type Getter interface {
	FromGitHubOpenPRs(ctx context.Context, rn *gh.RepoName) (CommitIntents, error)
	FromGitHubIssues(ctx context.Context, rn *gh.RepoName) (CommitIntents, error)
	FromLocalGitRepo(ctx context.Context, repo *git.Repository, from plumbing.Hash, since *time.Time) (CommitIntents, error)
}

type GetterImpl struct {
	finder markup.Finder
	gc     *github.Client
	logger logr.Logger
}

func NewIntentsGetter(finder markup.Finder, gc *github.Client, logger logr.Logger) *GetterImpl {
	return &GetterImpl{finder: finder, gc: gc, logger: logger}
}

func (g *GetterImpl) FromGitHubOpenPRs(ctx context.Context, rn *gh.RepoName) (CommitIntents, error) {
	intents := make(CommitIntents)

	opt := &github.PullRequestListOptions{State: "open"}

	for {
		prs, resp, err := g.gc.PullRequests.List(ctx, rn.Owner, rn.Repo, opt)
		if err != nil {
			return nil, fmt.Errorf("error while listing PRs: %v", err)
		}

		for _, pr := range prs {
			url := *pr.HTMLURL

			logger := g.logger.WithValues("url", url)
			logger.Info("Processing PR")

			if pr.Body == nil {
				logger.Info("PR body empty; skipping")
				continue
			}

			shas, err := g.finder.FindSHAs(*pr.Body)
			if err != nil {
				return nil, fmt.Errorf("error while looking for SHAs in %q: %v", *pr.Body, err)
			}

			for _, s := range shas {
				logger.Info("Adding SHA", "sha", s)
				intents[s] = *pr.HTMLURL
			}
		}

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return intents, nil
}

func (g *GetterImpl) FromGitHubIssues(ctx context.Context, rn *gh.RepoName) (CommitIntents, error) {
	intents := make(CommitIntents)

	opt := &github.IssueListByRepoOptions{
		Labels: []string{internal.GitStreamLabel},
		State:  "all",
	}

	for {
		issues, resp, err := g.gc.Issues.ListByRepo(ctx, rn.Owner, rn.Repo, opt)
		if err != nil {
			return nil, fmt.Errorf("error while listing issues: %v", err)
		}

		for _, issue := range issues {
			url := *issue.HTMLURL

			logger := g.logger.WithValues("url", url)
			logger.Info("Processing issue")

			if issue.Body == nil {
				logger.Info("Issue body empty; skipping")
				continue
			}

			shas, err := g.finder.FindSHAs(*issue.Body)
			if err != nil {
				return nil, fmt.Errorf("error while looking for SHAs in %q: %v", *issue.Body, err)
			}

			for _, s := range shas {
				logger.Info("Adding SHA", "SHA", s)
				intents[s] = url
			}
		}

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return intents, nil
}

func (g *GetterImpl) FromLocalGitRepo(ctx context.Context, repo *git.Repository, from plumbing.Hash, since *time.Time) (CommitIntents, error) {
	lo := git.LogOptions{
		From:  from,
		Since: since,
	}

	iter, err := repo.Log(&lo)
	if err != nil {
		return nil, fmt.Errorf("could not get an iterator on the downstream repo: %v", err)
	}

	intents := make(CommitIntents)

	err = iter.ForEach(func(commit *object.Commit) error {
		hash := commit.Hash

		logger := g.logger.WithValues("commit", hash)
		logger.Info("Processing commit")

		shas, err := g.finder.FindSHAs(commit.Message)
		if err != nil {
			return fmt.Errorf("error while finding SHAs in commit %s: %v", hash, err)
		}

		for _, s := range shas {
			logger.Info("Adding SHA", "sha", s)
			intents[s] = "commit " + hash.String()
		}

		return nil
	})

	return intents, err
}
