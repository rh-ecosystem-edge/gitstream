package github

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"text/template"

	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/go-github/v47/github"
	"github.com/qbarrand/gitstream/internal"
)

var (
	//go:embed templates/*
	tmplFS embed.FS

	templates = template.Must(
		template.ParseFS(tmplFS, "templates/*.tmpl"),
	)
)

type Creator interface {
	CreateIssue(ctx context.Context, gitErr *ProcessError, downstreamRepo *RepoName, upstreamURL string, commit *object.Commit) (*github.Issue, error)
	CreatePR(ctx context.Context, repoName *RepoName, branch, base, upstreamURL string, commit *object.Commit) (*github.PullRequest, error)
}

type CreatorImpl struct {
	gc *github.Client
}

func NewCreator(gc *github.Client) *CreatorImpl {
	return &CreatorImpl{gc: gc}
}

func (c *CreatorImpl) CreateIssue(
	ctx context.Context,
	gitErr *ProcessError,
	downstreamRepo *RepoName,
	upstreamURL string,
	commit *object.Commit,
) (*github.Issue, error) {
	sha := commit.Hash.String()

	data := IssueData{
		BaseData: BaseData{
			AppName:     internal.AppName,
			Commit:      Commit{SHA: sha},
			UpstreamURL: upstreamURL,
		},
		Error: *gitErr,
	}

	var buf bytes.Buffer

	if err := templates.ExecuteTemplate(&buf, "issue.tmpl", data); err != nil {
		return nil, fmt.Errorf("could not execute issue template: %v", err)
	}

	req := github.IssueRequest{
		Title: github.String(
			fmt.Sprintf("Cherry-picking error for `%s`", sha),
		),
		Body: github.String(
			buf.String(),
		),
		Labels: &[]string{internal.GitStreamLabel},
	}

	issue, _, err := c.gc.Issues.Create(ctx, downstreamRepo.Owner, downstreamRepo.Repo, &req)
	if err != nil {
		return nil, fmt.Errorf("could not create the issue: %v", err)
	}

	return issue, err
}

func (c *CreatorImpl) CreatePR(ctx context.Context, repoName *RepoName, branch, base, upstreamURL string, commit *object.Commit) (*github.PullRequest, error) {
	sha := commit.Hash.String()

	data := PRData{
		AppName:     internal.AppName,
		Commit:      Commit{SHA: sha},
		UpstreamURL: upstreamURL,
	}

	var buf bytes.Buffer

	if err := templates.ExecuteTemplate(&buf, "pr.tmpl", data); err != nil {
		return nil, fmt.Errorf("could not execute template: %v", err)
	}

	req := github.NewPullRequest{
		Title: github.String(
			fmt.Sprintf("Cherry-pick `%s` from upstream", sha),
		),
		Body: github.String(
			buf.String(),
		),
		Head: github.String(branch),
		Base: github.String(base),
	}

	pr, _, err := c.gc.PullRequests.Create(ctx, repoName.Owner, repoName.Repo, &req)
	if err != nil {
		return nil, fmt.Errorf("could not create the pull request: %v", err)
	}

	_, _, err = c.gc.Issues.AddLabelsToIssue(ctx, repoName.Owner, repoName.Repo, *pr.Number, []string{internal.GitStreamLabel})
	if err != nil {
		return nil, fmt.Errorf("could not label PR: %v", err)
	}

	return pr, err
}
