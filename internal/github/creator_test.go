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

func TestNewCreator(t *testing.T) {
	assert.NotNil(
		t,
		gh.NewCreator(nil),
	)
}

func TestCreatorImpl_CreateIssue(t *testing.T) {
	const expectedBody = "gitstream tried to cherry-pick commit `e3229f3c533ed51070beff092e5c7694a8ee81f0` from `some-upstream-url` but was unable to do so.\n" +
		"Please cherry-pick the commit manually.\n\n" +
		"---\n" +
		"**Return code:** 123\n\n" +
		"<details><summary>Output</summary>\n\n" +
		"```\n" +
		"test output\n" +
		"```\n\n" +
		"</details>\n\n" +
		"---\n\n" +
		"Upstream-Commit: e3229f3c533ed51070beff092e5c7694a8ee81f0"

	const expectedTitle = "Cherry-picking error for `e3229f3c533ed51070beff092e5c7694a8ee81f0`"

	issue := &github.Issue{Number: github.Int(456)}

	c := mock.NewMockedHTTPClient(
		mock.WithRequestMatchHandler(
			mock.PostReposIssuesByOwnerByRepo,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				m := make(map[string]interface{})

				assert.NoError(
					t,
					json.NewDecoder(r.Body).Decode(&m),
				)

				assert.Equal(t, expectedBody, m["body"])
				assert.Equal(t, expectedTitle, m["title"])
				assert.Contains(t, m["labels"], "gitstream")

				assert.NoError(
					t,
					json.NewEncoder(w).Encode(issue),
				)
			}),
		),
	)

	gc := github.NewClient(c)

	res, err := gh.NewCreator(gc).CreateIssue(
		context.Background(),
		&gh.ProcessError{Output: "test output", ReturnCode: github.Int(123)},
		&gh.RepoName{Owner: "owner", Repo: "repo"},
		"some-upstream-url",
		&object.Commit{Hash: plumbing.NewHash("e3229f3c533ed51070beff092e5c7694a8ee81f0")},
	)

	assert.NoError(t, err)
	assert.Equal(t, issue, res)
}

func TestCreatorImpl_CreatePR(t *testing.T) {
	const expectedBody = "This is an automated cherry-pick by gitstream of `e3229f3c533ed51070beff092e5c7694a8ee81f0` from `some-upstream-url`.\n\n" +
		"---\n\n" +
		"Upstream-Commit: e3229f3c533ed51070beff092e5c7694a8ee81f0"

	const expectedTitle = "Cherry-pick `e3229f3c533ed51070beff092e5c7694a8ee81f0` from upstream"

	const (
		owner    = "owner"
		repo     = "repo"
		prNumber = 456
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

	res, err := gh.NewCreator(gc).CreatePR(
		context.Background(),
		&gh.RepoName{Owner: owner, Repo: repo},
		"some-branch",
		"main",
		"some-upstream-url",
		&object.Commit{Hash: plumbing.NewHash("e3229f3c533ed51070beff092e5c7694a8ee81f0")},
	)

	assert.NoError(t, err)
	assert.Equal(t, pr, res)
}
