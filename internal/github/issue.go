package github

import (
	"bytes"
	"context"
	"fmt"

	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/go-github/v47/github"
	"github.com/rh-ecosystem-edge/gitstream/internal"
)

//go:generate mockgen -source=issue.go -package=github -destination=mock_issue.go

type IssueHelper interface {
	Create(ctx context.Context, err error, upstreamURL string, commit *object.Commit) (*github.Issue, error)
	ListAllOpen(ctx context.Context, includePRs bool) ([]*github.Issue, error)
	Assign(ctx context.Context, issue *github.Issue, usersLogin ...string) error
}

type IssueHelperImpl struct {
	gc       *github.Client
	markup   string
	repoName *RepoName
}

func NewIssueHelper(gc *github.Client, markup string, name *RepoName) *IssueHelperImpl {
	return &IssueHelperImpl{
		gc:       gc,
		markup:   markup,
		repoName: name,
	}
}

func (ih *IssueHelperImpl) Create(ctx context.Context, err error, upstreamURL string, commit *object.Commit) (*github.Issue, error) {
	sha := commit.Hash.String()

	data := IssueData{
		BaseData: BaseData{
			AppName: internal.AppName,
			Commit: Commit{
				Message: commit.Message,
				SHA:     sha,
			},
			JobID:       GetJobID(),
			Markup:      ih.markup,
			UpstreamURL: upstreamURL,
		},
		Error: err,
	}

	var buf bytes.Buffer

	if err := templates.ExecuteTemplate(&buf, "issue.tmpl", &data); err != nil {
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

	issue, _, err := ih.gc.Issues.Create(ctx, ih.repoName.Owner, ih.repoName.Repo, &req)
	if err != nil {
		return nil, fmt.Errorf("could not create the issue: %v", err)
	}

	return issue, err
}

func (ih *IssueHelperImpl) ListAllOpen(ctx context.Context, includePRs bool) ([]*github.Issue, error) {
	i := make([]*github.Issue, 0)

	opts := &github.IssueListByRepoOptions{
		Labels: []string{internal.GitStreamLabel},
		State:  "open",
	}

	for {
		issues, res, err := ih.gc.Issues.ListByRepo(ctx, ih.repoName.Owner, ih.repoName.Repo, opts)
		if err != nil {
			return nil, fmt.Errorf("could not list issues: %v", err)
		}

		for _, issue := range issues {
			if issue.PullRequestLinks != nil && !includePRs {
				continue
			}

			i = append(i, issue)
		}

		if res.NextPage == 0 {
			break
		}

		opts.Page = res.NextPage
	}

	return i, nil
}

func (ih *IssueHelperImpl) Assign(ctx context.Context, issue *github.Issue, usersLogin ...string) error {

	if _, _, err := ih.gc.Issues.AddAssignees(ctx, ih.repoName.Owner, ih.repoName.Repo, *issue.Number, usersLogin); err != nil {
		return fmt.Errorf("failed to add assignees: %v", err)
	}

	return nil
}
