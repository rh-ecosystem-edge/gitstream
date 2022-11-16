package github_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"testing"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/go-github/v47/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	gh "github.com/qbarrand/gitstream/internal/github"
	"github.com/qbarrand/gitstream/internal/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCreator(t *testing.T) {
	assert.NotNil(
		t,
		gh.NewCreator(nil, "", nil),
	)
}

func TestCreatorImpl_CreateIssue(t *testing.T) {
	const expectedTitle = "Cherry-picking error for `e3229f3c533ed51070beff092e5c7694a8ee81f0`"

	issue := &github.Issue{Number: github.Int(456)}
	repoName := &gh.RepoName{Owner: "owner", Repo: "repo"}

	t.Run("regular error", func(t *testing.T) {
		const expectedBody = "gitstream tried to cherry-pick commit `e3229f3c533ed51070beff092e5c7694a8ee81f0` from `some-upstream-url` but was unable to do so.\n" +
			"Please cherry-pick the commit manually.\n\n" +
			"---\n\n" +
			"**Error**:\n" +
			"```\n" +
			"random error\n" +
			"```\n\n\n" +
			"---\n\n" +
			"Markup: e3229f3c533ed51070beff092e5c7694a8ee81f0"

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

		res, err := gh.NewCreator(gc, "Markup", repoName).CreateIssue(
			context.Background(),
			errors.New("random error"),
			"some-upstream-url",
			&object.Commit{Hash: plumbing.NewHash("e3229f3c533ed51070beff092e5c7694a8ee81f0")},
		)

		assert.NoError(t, err)
		assert.Equal(t, issue, res)
	})

	t.Run("process error", func(t *testing.T) {
		const expectedBody = "gitstream tried to cherry-pick commit `e3229f3c533ed51070beff092e5c7694a8ee81f0` from `some-upstream-url` but was unable to do so.\n" +
			"Please cherry-pick the commit manually.\n\n" +
			"---\n\n" +
			"**Error**:\n" +
			"```\n" +
			"signal: killed\n" +
			"```\n" +
			"---\n\n" +
			"**Command**: `some-command`\n\n" +
			"<details><summary>Output</summary>\n\n" +
			"```\n" +
			"some output\n" +
			"```\n\n" +
			"</details>\n\n\n" +
			"---\n\n" +
			"Other-Markup: e3229f3c533ed51070beff092e5c7694a8ee81f0"

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

		ctx2, cancel := context.WithCancel(context.Background())

		cmd := exec.CommandContext(ctx2, "cat")
		err := cmd.Start()

		require.NoError(t, err)

		cancel()

		ee := &exec.ExitError{}

		assert.ErrorAs(t, cmd.Wait(), &ee)

		res, err := gh.NewCreator(gc, "Other-Markup", repoName).CreateIssue(
			context.Background(),
			process.NewError(ee, []byte("some output"), "some-command"),
			"some-upstream-url",
			&object.Commit{Hash: plumbing.NewHash("e3229f3c533ed51070beff092e5c7694a8ee81f0")},
		)

		assert.NoError(t, err)
		assert.Equal(t, issue, res)
	})
}

func TestCreatorImpl_CreatePR(t *testing.T) {
	const (
		draft        = true
		expectedBody = "This is an automated cherry-pick by gitstream of `e3229f3c533ed51070beff092e5c7694a8ee81f0` from `some-upstream-url`.\n\n" +
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

	res, err := gh.NewCreator(gc, "Markup", &gh.RepoName{Owner: owner, Repo: repo}).CreatePR(
		context.Background(),
		"some-branch",
		"main",
		"some-upstream-url",
		&object.Commit{Hash: plumbing.NewHash("e3229f3c533ed51070beff092e5c7694a8ee81f0")},
		draft,
	)

	assert.NoError(t, err)
	assert.Equal(t, pr, res)
}
