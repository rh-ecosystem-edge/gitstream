package gitstream

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-logr/logr"
	"github.com/qbarrand/gitstream/internal"
	"github.com/qbarrand/gitstream/internal/config"
	gh "github.com/qbarrand/gitstream/internal/github"
	"github.com/qbarrand/gitstream/internal/gitutils"
	"github.com/qbarrand/gitstream/internal/process"
)

type Sync struct {
	CherryPicker     gitutils.CherryPicker
	Differ           gitutils.Differ
	DiffConfig       config.Diff
	DownstreamConfig config.Downstream
	DryRun           bool
	GitHelper        gitutils.Helper
	GitHubToken      string
	IssueHelper      gh.IssueHelper
	Logger           logr.Logger
	PRHelper         gh.PRHelper
	Repo             *git.Repository
	RepoName         *gh.RepoName
	UpstreamConfig   config.Upstream
}

func (s *Sync) Run(ctx context.Context) error {
	commits, err := s.Differ.GetMissingCommits(
		ctx,
		s.Repo,
		s.RepoName,
		s.DiffConfig.CommitsSince,
		s.DownstreamConfig.MainBranch,
		s.UpstreamConfig,
	)
	if err != nil {
		return fmt.Errorf("could not get commits not present in downstream: %v", err)
	}

	s.Logger.V(1).Info("Listing GitStream issues (including PRs)")

	issuesAndPRs, err := s.IssueHelper.ListAllOpen(ctx, true)
	if err != nil {
		return fmt.Errorf("could not list issues: %v", err)
	}

	existingOpenIssues := len(issuesAndPRs)

	s.Logger.V(1).Info("Listed GitStream issues (including PRs)", "total", existingOpenIssues)

	maxItems := s.DownstreamConfig.MaxOpenItems

	if maxItems != -1 && existingOpenIssues > maxItems {
		s.Logger.Info(
			"Maximum number of items on GitHub exceeded",
			"open", existingOpenIssues,
			"max", maxItems,
		)

		return nil
	}

	sort.Slice(commits, func(i, j int) bool {
		return commits[i].Committer.When.Before(commits[j].Committer.When)
	})

	wt, err := s.Repo.Worktree()
	if err != nil {
		return fmt.Errorf("could not get the worktree: %v", err)
	}

	mainCheckoutOptions := git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(s.DownstreamConfig.MainBranch),
		Force:  true,
	}

	canBeCreated := maxItems - existingOpenIssues

	for _, c := range commits {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if maxItems != -1 && canBeCreated <= 0 {
			s.Logger.Info(
				"Maximum number of open objects reached",
				"existing", existingOpenIssues,
				"max", maxItems,
			)

			return nil
		}

		canBeCreated--

		sha := c.Hash.String()

		logger := s.Logger.WithValues("sha", sha)

		logger.Info("Cherry-picking commit")

		logger.Info("Checking out main branch", "name", s.DownstreamConfig.MainBranch)

		if err := wt.Checkout(&mainCheckoutOptions); err != nil {
			return fmt.Errorf("could not checkout the main branch: %v", err)
		}

		if err := wt.Reset(&git.ResetOptions{Mode: git.HardReset}); err != nil {
			return fmt.Errorf("could not reset: %v", err)
		}

		branchName := internal.GitStreamPrefix + sha

		logger.Info("Switching to branch", "name", branchName)

		branchRef := plumbing.NewBranchReferenceName(branchName)

		if err := s.Repo.Storer.RemoveReference(branchRef); err != nil {
			return fmt.Errorf("could not remove reference %q for branch %s: %v", branchRef, branchName, err)
		}

		co := git.CheckoutOptions{
			Branch: branchRef,
			Create: true,
			Force:  true,
		}

		if err := wt.Checkout(&co); err != nil {
			return fmt.Errorf("could not checkout branch %s: %v", branchName, err)
		}

		logger.Info("Running cherry-pick")

		if err := s.cherryPickAndPush(ctx, c, branchName, logger); err != nil {
			if s.DryRun {
				logger.Info("Dry run: skipping issue creation")
				continue
			}

			if issue, err := s.IssueHelper.Create(ctx, err, s.UpstreamConfig.URL, c); err != nil {
				logger.Error(err, "could not create issue for commit")
			} else {
				logger.Info("Created issue", "url", *issue.HTMLURL)
			}
		}
	}

	return nil
}

func (s *Sync) cherryPickAndPush(ctx context.Context, commit *object.Commit, branchName string, logger logr.Logger) error {
	if err := s.CherryPicker.Run(ctx, s.Repo, s.DownstreamConfig.LocalRepoPath, commit); err != nil {
		pe := &process.Error{}

		if errors.As(err, &pe) {
			logger.Info("Output", "combined", pe.CombinedString())
		}

		return fmt.Errorf("could not cherry-pick: %w", err)
	}

	if s.DryRun {
		logger.Info("Dry run: skipping push")
		return nil
	}

	if err := s.GitHelper.PushContextWithAuth(ctx, s.GitHubToken, "origin"); err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return fmt.Errorf("error while pushing branch %s: %v", branchName, err)
	}

	pr, err := s.PRHelper.Create(
		ctx,
		branchName,
		s.DownstreamConfig.MainBranch,
		s.UpstreamConfig.URL,
		commit,
		s.DownstreamConfig.CreateDraftPRs,
	)

	if err != nil {
		return fmt.Errorf("could not create PR: %v", err)
	}

	logger.Info("Created PR", "url", pr.HTMLURL)

	return nil
}
