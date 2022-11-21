package github_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/go-github/v47/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	gh "github.com/qbarrand/gitstream/internal/github"
	"github.com/stretchr/testify/assert"
)

func TestPRHelperImpl_Create(t *testing.T) {
	const (
		draft        = true
		expectedBody = "This is an automated cherry-pick by gitstream of `e3229f3c533ed51070beff092e5c7694a8ee81f0` from `some-upstream-url`.\n\n" +
			"Commit message:\n" +
			"```\n" +
			"Some commit message\n" +
			"spreading over two lines.\n" +
			"```\n\n" +
			"---\n\n" +
			"Markup: e3229f3c533ed51070beff092e5c7694a8ee81f0"
		expectedTitle = "Cherry-pick `e3229f3c533ed51070beff092e5c7694a8ee81f0` from upstream"
		owner         = "owner"
		prNumber      = 456
		repo          = "repo"
	)

	pr := &github.PullRequest{
		Number: github.Int(prNumber),
	}

	c := mock.NewMockedHTTPClient(
		mock.WithRequestMatchHandler(
			mock.PostReposPullsByOwnerByRepo,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				m := make(map[string]interface{})

				assert.NoError(
					t,
					json.NewDecoder(r.Body).Decode(&m),
				)

				assert.Equal(t, expectedBody, m["body"])
				assert.Equal(t, expectedTitle, m["title"])
				assert.Equal(t, draft, m["draft"])

				assert.NoError(
					t,
					json.NewEncoder(w).Encode(pr),
				)
			}),
		),
		mock.WithRequestMatchHandler(
			mock.PostReposIssuesLabelsByOwnerByRepoByIssueNumber,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				m := make([]string, 0)

				assert.NoError(
					t,
					json.NewDecoder(r.Body).Decode(&m),
				)

				assert.Equal(
					t,
					fmt.Sprintf("/repos/%s/%s/issues/%d/labels", owner, repo, prNumber),
					r.RequestURI,
				)

				assert.Equal(t, []string{"gitstream"}, m)
			}),
		),
	)

	gc := github.NewClient(c)

	res, err := gh.NewPRHelper(gc, nil, "Markup", &gh.RepoName{Owner: owner, Repo: repo}).Create(
		context.Background(),
		"some-branch",
		"main",
		"some-upstream-url",
		&object.Commit{
			Hash:    plumbing.NewHash("e3229f3c533ed51070beff092e5c7694a8ee81f0"),
			Message: "Some commit message\nspreading over two lines.",
		},
		draft,
	)

	assert.NoError(t, err)
	assert.Equal(t, pr, res)
}
