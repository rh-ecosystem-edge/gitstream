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
	DryRun         bool
	Finder         markup.Finder
	GitHelper      gitutils.Helper
	Logger         logr.Logger
	PRHelper       gh.PRHelper
	Repo           *git.Repository
	RepoName       *gh.RepoName
	UpstreamConfig config.Upstream
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

	prs, err := u.PRHelper.ListAllOpen(ctx, nil)
	if err != nil {
		return fmt.Errorf("could not list open PRs: %v", err)
	}

	for _, pr := range prs {
		logger := u.Logger.WithValues("url", *pr.HTMLURL)
		logger.Info("Processing PR")

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

	if oldestGitStreamPR == nil {
		u.Logger.Info("No GitStream PR found")
		return nil
	}

	logger := u.Logger.WithValues("url", oldestGitStreamPR.HTMLURL)

	if !*oldestGitStreamPR.Draft {
		logger.Info("Oldest PR is not a draft; exiting")
		return nil
	}

	logger.Info("Making PR ready for review")

	if u.DryRun {
		logger.Info("Dry run: skipping PR update")
		return nil
	}

	if err := u.PRHelper.MakeReady(ctx, oldestGitStreamPR); err != nil {
		return fmt.Errorf("could not mark PR %d ready for review: %v", *oldestGitStreamPR.Number, err)
	}

	return nil
}
