package gitstream

import (
	"context"
	"fmt"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-logr/logr"
	"github.com/google/go-github/v47/github"
	"github.com/qbarrand/gitstream/internal"
	"github.com/qbarrand/gitstream/internal/config"
	gh "github.com/qbarrand/gitstream/internal/github"
	"github.com/qbarrand/gitstream/internal/gitutils"
	"github.com/qbarrand/gitstream/internal/markup"
)

type Undraft struct {
	DiffConfig       config.Diff
	DownstreamConfig config.Downstream
	DryRun           bool
	Finder           markup.Finder
	GitHelper        gitutils.Helper
	GitHubClient     *github.Client
	Logger           logr.Logger
	Repo             *git.Repository
	RepoName         *gh.RepoName
	UpstreamConfig   config.Upstream
}

func (u *Undraft) Run(ctx context.Context) error {
	const remoteName = internal.UpstreamRemoteName

	if _, err := u.GitHelper.RecreateRemote(ctx, remoteName, u.UpstreamConfig.URL); err != nil {
		return fmt.Errorf("could not recreate remote: %v", err)
	}

	if err := u.GitHelper.FetchRemoteContext(ctx, remoteName); err != nil {
		return fmt.Errorf("could not fetch remote %s: %v", remoteName, err)
	}

	var (
		oldestTime        *time.Time
		oldestGitStreamPR *github.PullRequest
	)

	opt := &github.PullRequestListOptions{State: "open"}

	for {
		prs, resp, err := u.GitHubClient.PullRequests.List(ctx, u.RepoName.Owner, u.RepoName.Repo, opt)
		if err != nil {
			return fmt.Errorf("error while listing PRs: %v", err)
		}

		for _, pr := range prs {
			url := *pr.HTMLURL

			logger := u.Logger.WithValues("url", url)
			logger.Info("Processing PR")

			if pr.Body == nil {
				logger.Info("PR body empty; skipping")
				continue
			}

			// go-github sadly does not support filtering PRs by label
			if label := internal.GitStreamLabel; !prHasLabel(pr, label) {
				logger.Info("Missing label; not a GitStream PR; skipping", "label", label)
			}

			shas, err := u.Finder.FindSHAs(*pr.Body)
			if err != nil {
				return fmt.Errorf("error while looking for SHAs in %q: %v", *pr.Body, err)
			}

			for _, s := range shas {
				upstreamCommit, err := u.Repo.CommitObject(s)
				if err != nil {
					return fmt.Errorf("could not find upstream commit %s: %v", s, err)
				}

				if t := upstreamCommit.Committer.When; oldestTime == nil || t.Before(*oldestTime) {
					oldestGitStreamPR = pr
				}

				logger.Info("Adding SHA", "sha", s)
			}
		}

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	if oldestGitStreamPR == nil {
		u.Logger.Info("No GitStream PR found")
		return nil
	}

	logger := u.Logger.WithValues("url", oldestGitStreamPR.HTMLURL)

	if !*oldestGitStreamPR.Draft {
		logger.Info("Oldest PR is not a draft; exiting")
		return nil
	}

	*oldestGitStreamPR.Draft = false

	logger.Info("Making PR ready for review")

	if u.DryRun {
		logger.Info("Dry run: skipping PR update")
	}

	if _, _, err := u.GitHubClient.PullRequests.Edit(ctx, u.RepoName.Owner, u.RepoName.Repo, *oldestGitStreamPR.Number, oldestGitStreamPR); err != nil {
		return fmt.Errorf("could not make PR %d ready for review: %v", *oldestGitStreamPR.Number, err)
	}

	return nil
}

func prHasLabel(pr *github.PullRequest, label string) bool {
	for _, l := range pr.Labels {
		if *l.Name == label {
			return true
		}
	}

	return false
}
