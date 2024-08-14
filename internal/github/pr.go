package github

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/cli/go-gh/pkg/api"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/go-github/v47/github"
	"github.com/rh-ecosystem-edge/gitstream/internal"
	"github.com/shurcooL/githubv4"
)

type PRFilterFunc = func(*github.PullRequest) bool

//go:generate mockgen -source=pr.go -package=github -destination=mock_pr.go

type PRHelper interface {
	Create(ctx context.Context, branch, base, upstreamURL string, commit *object.Commit, draft bool) (*github.PullRequest, error)
	ListAllOpen(ctx context.Context, filter PRFilterFunc) ([]*github.PullRequest, error)
	MakeReady(ctx context.Context, pr *github.PullRequest) error
}

type PRHelperImpl struct {
	gc       *github.Client
	ghgql    api.GQLClient
	markup   string
	repoName *RepoName
}

func NewPRHelper(gc *github.Client, ghgql api.GQLClient, markup string, repoName *RepoName) *PRHelperImpl {
	return &PRHelperImpl{
		gc:       gc,
		ghgql:    ghgql,
		markup:   markup,
		repoName: repoName,
	}
}

func (ph *PRHelperImpl) Create(ctx context.Context, branch, base, upstreamURL string, commit *object.Commit, draft bool) (*github.PullRequest, error) {
	sha := commit.Hash.String()

	data := PRData{
		AppName: internal.AppName,
		Commit: Commit{
			Message: commit.Message,
			SHA:     sha,
		},
		Markup:      ph.markup,
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
		Head:  github.String(branch),
		Base:  github.String(base),
		Draft: &draft,
	}

	pr, _, err := ph.gc.PullRequests.Create(ctx, ph.repoName.Owner, ph.repoName.Repo, &req)
	if err != nil {
		return nil, fmt.Errorf("could not create the pull request: %v", err)
	}

	_, _, err = ph.gc.Issues.AddLabelsToIssue(ctx, ph.repoName.Owner, ph.repoName.Repo, *pr.Number, []string{internal.GitStreamLabel})
	if err != nil {
		return nil, fmt.Errorf("could not label PR: %v", err)
	}

	return pr, nil
}

func (ph *PRHelperImpl) ListAllOpen(ctx context.Context, filter PRFilterFunc) ([]*github.PullRequest, error) {
	p := make([]*github.PullRequest, 0)

	opts := &github.PullRequestListOptions{State: "open"}

	for {
		prs, res, err := ph.gc.PullRequests.List(ctx, ph.repoName.Owner, ph.repoName.Repo, opts)
		if err != nil {
			return nil, fmt.Errorf("could not list PRs: %v", err)
		}

		for _, pr := range prs {
			if label := internal.GitStreamLabel; !PRHasLabel(pr, label) {
				continue
			}

			if filter != nil && !filter(pr) {
				continue
			}

			p = append(p, pr)
		}

		if res.NextPage == 0 {
			break
		}

		opts.Page = res.NextPage
	}

	return p, nil
}

func (ph *PRHelperImpl) MakeReady(ctx context.Context, pr *github.PullRequest) error {
	if !*pr.Draft {
		return errors.New("PR is not a draft")
	}

	// Use GraphQL as the REST API does not support making a PR ready for review
	var mutation struct {
		MarkPullRequestReadyForReview struct {
			PullRequest struct {
				ID githubv4.ID
			}
		} `graphql:"markPullRequestReadyForReview(input: $input)"`
	}

	variables := map[string]interface{}{
		"input": githubv4.MarkPullRequestReadyForReviewInput{
			PullRequestID: pr.NodeID,
		},
	}

	return ph.ghgql.MutateWithContext(ctx, "PullRequestReadyForReview", &mutation, variables)
}

func PRHasLabel(pr *github.PullRequest, label string) bool {
	for _, l := range pr.Labels {
		if *l.Name == label {
			return true
		}
	}

	return false
}
