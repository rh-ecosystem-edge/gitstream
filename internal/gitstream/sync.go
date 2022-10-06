package gitstream

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-logr/logr"
	"github.com/qbarrand/gitstream/internal"
	gh "github.com/qbarrand/gitstream/internal/github"
	"github.com/qbarrand/gitstream/internal/gitutils"
)

type Sync struct {
	Creator                 gh.Creator
	Differ                  gitutils.Differ
	DryRun                  bool
	DownstreamIntentsGetter gitutils.DownstreamIntentsGetter
	Repo                    *gitutils.RepoWrapper
	DownstreamRepoName      *gh.RepoName
	GitHelper               gitutils.Helper
	Logger                  logr.Logger
	UpstreamSince           *time.Time
	UpstreamRef             string
	UpstreamURL             string
}

const mainBranch = "main"

func (s *Sync) Run(ctx context.Context) error {
	downstreamIntents, err := s.DownstreamIntentsGetter.GetIntents(ctx, s.Repo.Repository, s.UpstreamSince, s.DownstreamRepoName)
	if err != nil {
		return fmt.Errorf("could not get downstream commit intents: %v", err)
	}

	if err := prepareRemote(ctx, s.Repo.Repository, s.UpstreamURL, s.GitHelper); err != nil {
		return fmt.Errorf("could not prepare the remote pointing to upstream: %v", err)
	}

	ref, err := getRefFromRemote(s.Repo.Repository, upstreamRemoteName, s.UpstreamRef)
	if err != nil {
		return fmt.Errorf("could not get the reference for %s/%s: %v", upstreamRemoteName, s.UpstreamRef, err)
	}

	diff, err := s.Differ.GetMissingCommits(ctx, s.Repo.Repository, ref, s.UpstreamSince, downstreamIntents)
	if err != nil {
		return fmt.Errorf("could not get commits not present in downstream: %v", err)
	}

	wt, err := s.Repo.Repository.Worktree()
	if err != nil {
		return fmt.Errorf("could not get the worktree: %v", err)
	}

	mainCheckoutOptions := git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(mainBranch),
		Force:  true,
	}

	for h, c := range diff {
		logger := s.Logger.WithValues("sha", h)
		logger.Info("Cherry-picking commit")

		logger.Info("Checking out main branch", "name", mainBranch)

		if err := wt.Checkout(&mainCheckoutOptions); err != nil {
			return fmt.Errorf("could not checkout the main branch: %v", err)
		}

		if err := wt.Reset(&git.ResetOptions{Mode: git.HardReset}); err != nil {
			return fmt.Errorf("could not reset: %v", err)
		}

		branchName := internal.GitStreamPrefix + h.String()

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

		out, err := s.Repo.CherryPick(ctx, c)
		if err != nil {
			logger.Info("Could not cherry-pick; creating issue", "output", string(out))

			exitErr := &exec.ExitError{}
			var exitCode *int

			if errors.As(err, &exitErr) {
				code := exitErr.ExitCode()
				exitCode = &code
			}

			logger.Error(err, "Error while cherry-picking; creating issue")

			pe := gh.ProcessError{
				Output:     string(out),
				ReturnCode: exitCode,
			}

			if s.DryRun {
				logger.Info("Dry run: skipping issue creation")
			} else {
				if issue, err := s.Creator.CreateIssue(ctx, &pe, s.DownstreamRepoName, s.UpstreamURL, c); err != nil {
					logger.Error(err, "could not create issue for commit")
				} else {
					logger.Info("Created issue", "url", *issue.HTMLURL)
				}
			}

			continue
		}

		if s.DryRun {
			logger.Info("Dry run: skipping push")
			continue
		}

		if err := s.Repo.PushContextWithAuth(ctx); err != nil {
			return fmt.Errorf("error while pushing branch %s: %v", branchName, err)
		}

		pr, err := s.Creator.CreatePR(ctx, s.DownstreamRepoName, branchName, mainBranch, s.UpstreamURL, c)
		if err != nil {
			return fmt.Errorf("could not create PR: %v", err)
		}

		logger.Info("Created PR", "url", pr.HTMLURL)
	}

	return nil
}
