package gitstream

import (
	"context"
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/go-logr/logr"
	"github.com/google/go-github/v47/github"
	"github.com/qbarrand/gitstream/internal"
	"github.com/qbarrand/gitstream/internal/config"
	gh "github.com/qbarrand/gitstream/internal/github"
	"github.com/qbarrand/gitstream/internal/gitutils"
	"github.com/qbarrand/gitstream/internal/markup"
	"gopkg.in/yaml.v3"
)

type Assign struct {
	GC             *github.Client
	DryRun         bool
	Finder         markup.Finder
	GitHelper      gitutils.Helper
	Logger         logr.Logger
	IssueHelper    gh.IssueHelper
	UserHelper     gh.UserHelper
	Repo           *git.Repository
	RepoName       *gh.RepoName
	UpstreamConfig config.Upstream
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

func (a *Assign) getOwnersContent(ctx context.Context) (*Owners, error) {

	const filePath = "OWNERS"

	content, _, _, err := a.GC.Repositories.GetContents(ctx, a.RepoName.Owner, a.RepoName.Repo, filePath, nil)
	if err != nil || content == nil { // `content` can be nil if the filePath refers to a directory
		return nil, fmt.Errorf("could not get %s's content from %s/%s: %v", filePath, a.RepoName.Owner, a.RepoName.Repo, err)
	}
	data, err := content.GetContent()
	if err != nil {
		return nil, fmt.Errorf("%s's data is invalid: %v", filePath, err)
	}

	var owners Owners
	if err := yaml.Unmarshal([]byte(data), &owners); err != nil {
		return nil, fmt.Errorf("could not unmarshal %s as a yaml: %v", filePath, err)
	}

	return &owners, nil
}

func (a *Assign) assignIssues(ctx context.Context) error {

	owners, err := a.getOwnersContent(ctx)
	if err != nil {
		return fmt.Errorf("could not get owners content: %v", err)
	}

	issues, err := a.IssueHelper.ListAllOpen(ctx, true)
	if err != nil {
		return fmt.Errorf("could not list open issues: %v", err)
	}

	for _, issue := range issues {

		logger := a.Logger.WithValues("url", *issue.HTMLURL)
		logger.Info("Processing issue")

		shas, err := a.Finder.FindSHAs(*issue.Body)
		if err != nil {
			return fmt.Errorf("error while looking for SHAs in %q: %v", *issue.Body, err)
		}

		for _, s := range shas {

			upstreamCommit, err := a.Repo.CommitObject(s)
			if err != nil {
				return fmt.Errorf("could not find upstream commit %s: %v", s, err)
			}

			if a.DryRun {
				logger.Info("Dry run: skipping issue update")
				return nil
			}

			logger.Info("Assigning issue")

			user, err := a.UserHelper.GetUser(ctx, upstreamCommit)
			if err != nil {
				return fmt.Errorf("could not get the upstream commit author for downstream issue %d: %v", *issue.Number, err)
			}

			assignee, err := owners.getAssignee(ctx, a.GC, *user.Login)
			if err != nil {
				return fmt.Errorf("could not get select a suitable assignee: %v", err)
			}

			if err := a.IssueHelper.Assign(ctx, issue, assignee); err != nil {
				return fmt.Errorf("could not assign issue %d: %v", *issue.Number, err)
			}
		}
	}

	return nil
}
