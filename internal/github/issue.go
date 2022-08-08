package github

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"text/template"

	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/go-github/v47/github"
	"github.com/qbarrand/gitstream/internal"
)

//go:embed templates/issue.tmpl
var issueTmpl []byte

var issueTemplate = template.Must(
	template.New("issue").Parse(string(issueTmpl)),
)

type IssueCreator interface {
	CreateIssue(ctx context.Context, gitErr error, output string) (*github.Issue, error)
}

type IssueCreatorImpl struct {
	gc *github.Client
}

func NewIssueCreator(gc *github.Client) IssueCreatorImpl {
	return IssueCreatorImpl{gc: gc}
}

func (c *IssueCreatorImpl) CreateIssue(ctx context.Context, processErr *ProcessError, repo *Repository, commit *object.Commit) (*github.Issue, error) {
	sha := commit.Hash.String()

	data := IssueData{
		AppName:  internal.AppName,
		Commit:   Commit{SHA: sha},
		Error:    *processErr,
		Upstream: *repo,
	}

	var buf bytes.Buffer

	if err := issueTemplate.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("could not execute issue template: %v", err)
	}

	title := fmt.Sprintf("Cherry-picking error for `%s`", sha)
	body := buf.String()

	req := github.IssueRequest{
		Title:  &title,
		Body:   &body,
		Labels: &[]string{internal.GitStreamLabel},
	}

	issue, _, err := c.gc.Issues.Create(ctx, repo.Name.Owner, repo.Name.Repo, &req)
	if err != nil {
		return nil, fmt.Errorf("could not create the issue: %v", err)
	}

	return issue, err
}
