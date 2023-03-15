package gitstream

import (
	"context"
	"errors"
	"fmt"
	"path"

	"github.com/go-git/go-git/v5"
	"github.com/go-logr/logr"
	"github.com/google/go-github/v47/github"
	"github.com/qbarrand/gitstream/internal"
	"github.com/qbarrand/gitstream/internal/config"
	gh "github.com/qbarrand/gitstream/internal/github"
	"github.com/qbarrand/gitstream/internal/gitutils"
	"github.com/qbarrand/gitstream/internal/markup"
	"github.com/qbarrand/gitstream/internal/owners"
)

type Assign struct {
	GC               *github.Client
	DryRun           bool
	Finder           markup.Finder
	GitHelper        gitutils.Helper
	Logger           logr.Logger
	IssueHelper      gh.IssueHelper
	UserHelper       gh.UserHelper
	Repo             *git.Repository
	RepoName         *gh.RepoName
	UpstreamConfig   config.Upstream
	DownstreamConfig config.Downstream
	OwnersHelper     owners.OwnersHelper
}

func (a *Assign) Run(ctx context.Context) error {
	const remoteName = internal.UpstreamRemoteName

	if _, err := a.GitHelper.RecreateRemote(ctx, remoteName, a.UpstreamConfig.URL); err != nil {
		return fmt.Errorf("could not recreate remote: %v", err)
	}

	if err := a.GitHelper.FetchRemoteContext(ctx, remoteName, a.UpstreamConfig.Ref); err != nil {
		return fmt.Errorf("could not fetch remote %s: %v", remoteName, err)
	}

	if err := a.assignIssues(ctx); err != nil {
		return fmt.Errorf("could not add assignees to issues: %v", err)
	}

	return nil
}

func (a *Assign) assignIssues(ctx context.Context) error {

	ownersFile := path.Join(a.DownstreamConfig.LocalRepoPath, a.DownstreamConfig.OwnersFile)
	owners, err := a.OwnersHelper.FromFile(ownersFile)
	if err != nil {
		return fmt.Errorf("could not get owners from file %s: %v", ownersFile, err)
	}

	issues, err := a.IssueHelper.ListAllOpen(ctx, true)
	if err != nil {
		return fmt.Errorf("could not list open issues: %v", err)
	}

	for _, issue := range issues {

		logger := a.Logger.WithValues("url", *issue.HTMLURL)

		if len(issue.Assignees) > 0 {
			continue
		}

		logger.Info("Processing issue")

		shas, err := a.Finder.FindSHAs(*issue.Body)
		if err != nil {
			return fmt.Errorf("error while looking for SHAs in %q: %v", *issue.Body, err)
		}

		for _, s := range shas {

			if a.DryRun {
				logger.Info("Dry run: skipping issue update")
				return nil
			}

			logger.Info("Assigning issue")

			var (
				assignee string
				intErr   error
			)
			if user, err := a.UserHelper.GetCommitAuthor(ctx, s.String()); err != nil {
				if !errors.Is(err, gh.ErrUnexpectedReply) {
					logger.Info("WARNING: failed to get a response from Github, skipping commit",
						"issue", *issue.Number, "error", err.Error())
					continue
				}
				logger.Info("WARNING: commit author for downstream issue not found on github, picking a random assignee",
					"issue", *issue.Number, "error", err.Error())
				assignee, intErr = a.OwnersHelper.GetRandomApprover(owners)
				if intErr != nil {
					return fmt.Errorf("could not get a random owner: %v", err)
				}
			} else {
				if a.OwnersHelper.IsApprover(owners, *user.Login) {
					assignee = *user.Login
				} else {
					logger.Info("commit author for downstream issue is not an owner, picking a random assignee",
						"issue", *issue.Number)
					assignee, intErr = a.OwnersHelper.GetRandomApprover(owners)
					if intErr != nil {
						return fmt.Errorf("could not get a random owner: %v", err)
					}
				}
			}

			if err := a.IssueHelper.Assign(ctx, issue, assignee); err != nil {
				return fmt.Errorf("could not assign issue %d: %v", *issue.Number, err)
			}
		}
	}

	return nil
}
