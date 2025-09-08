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
	gh "github.com/rh-ecosystem-edge/gitstream/internal/github"
	"github.com/rh-ecosystem-edge/gitstream/internal/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssueHelper_Create(t *testing.T) {
	const expectedTitle = "Cherry-picking error for `e3229f3c533ed51070beff092e5c7694a8ee81f0`"

	issue := &github.Issue{Number: github.Int(456)}
	repoName := &gh.RepoName{Owner: "owner", Repo: "repo"}

	commit := &object.Commit{
		Hash:    plumbing.NewHash("e3229f3c533ed51070beff092e5c7694a8ee81f0"),
		Message: "Some commit message\nspanning over two lines.",
	}

	t.Run("regular error", func(t *testing.T) {
		const expectedBody = "gitstream tried to cherry-pick commit `e3229f3c533ed51070beff092e5c7694a8ee81f0` from `some-upstream-url` but was unable to do so.\n" +
			"\n" +
			"Commit message:\n" +
			"```\n" +
			"Some commit message\n" +
			"spanning over two lines.\n" +
			"```\n\n" +
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

		res, err := gh.NewIssueHelper(gc, "Markup", repoName).Create(
			context.Background(),
			errors.New("random error"),
			"some-upstream-url",
			commit,
		)

		assert.NoError(t, err)
		assert.Equal(t, issue, res)
	})

	t.Run("process error", func(t *testing.T) {
		const bodyFmt = "gitstream tried to cherry-pick commit `e3229f3c533ed51070beff092e5c7694a8ee81f0` from `some-upstream-url` but was unable to do so.\n" +
			"\n" +
			"Commit message:\n" +
			"```\n" +
			"Some commit message\n" +
			"spanning over two lines.\n" +
			"```\n\n" +
			"Please cherry-pick the commit manually.\n\n" +
			"---\n\n" +
			"**Error**:\n" +
			"```\n" +
			"%s\n" +
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

		bodies := []string{
			fmt.Sprintf(bodyFmt, "signal: killed"),
			fmt.Sprintf(bodyFmt, "<nil>"),
		}

		c := mock.NewMockedHTTPClient(
			mock.WithRequestMatchHandler(
				mock.PostReposIssuesByOwnerByRepo,
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					m := make(map[string]interface{})

					assert.NoError(
						t,
						json.NewDecoder(r.Body).Decode(&m),
					)

					assert.Contains(t, bodies, m["body"])
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

		res, err := gh.NewIssueHelper(gc, "Other-Markup", repoName).Create(
			context.Background(),
			process.NewError(ee, []byte("some output"), "some-command"),
			"some-upstream-url",
			commit,
		)

		assert.NoError(t, err)
		assert.Equal(t, issue, res)
	})
}

func TestIssueHelper_Assign(t *testing.T) {

	issue := &github.Issue{Number: github.Int(456)}
	username := "test-user"
	repoName := &gh.RepoName{Owner: "owner", Repo: "repo"}

	t.Run("API error", func(t *testing.T) {

		c := mock.NewMockedHTTPClient(
			mock.WithRequestMatchHandler(
				mock.PostReposIssuesAssigneesByOwnerByRepoByIssueNumber,
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					m := make(map[string]interface{})
					assert.NoError(
						t,
						json.NewDecoder(r.Body).Decode(&m),
					)
					assert.Equal(t, []interface{}{username}, m["assignees"])

					w.WriteHeader(http.StatusBadRequest)
				}),
			),
		)

		gc := github.NewClient(c)

		err := gh.NewIssueHelper(gc, "Markup", repoName).Assign(context.Background(), issue, username)

		assert.Error(t, err)
		assert.ErrorContains(t, err, "failed to add assignees")
	})

	t.Run("working as expected", func(t *testing.T) {

		username2 := "test-user-2"

		c := mock.NewMockedHTTPClient(
			mock.WithRequestMatchHandler(
				mock.PostReposIssuesAssigneesByOwnerByRepoByIssueNumber,
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					m := make(map[string]interface{})
					assert.NoError(
						t,
						json.NewDecoder(r.Body).Decode(&m),
					)
					assert.Equal(t, []interface{}{username, username2}, m["assignees"])
					assert.NoError(
						t,
						json.NewEncoder(w).Encode(issue),
					)
				}),
			),
		)

		gc := github.NewClient(c)

		err := gh.NewIssueHelper(gc, "Markup", repoName).Assign(context.Background(), issue, username, username2)

		assert.NoError(t, err)
	})
}

func TestIssueHelper_Comment(t *testing.T) {

	const (
		repoOwner = "owner"
		repoName  = "repo"
	)

	var (
		issueNumber = 123
		commentBody = "This is a test comment"

		issue = &github.Issue{
			Number: &issueNumber,
		}

		ghRepoName = &gh.RepoName{
			Owner: repoOwner,
			Repo:  repoName,
		}
	)

	t.Run("API error", func(t *testing.T) {
		c := mock.NewMockedHTTPClient(
			mock.WithRequestMatchHandler(
				mock.PostReposIssuesCommentsByOwnerByRepoByIssueNumber,
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				}),
			),
		)

		gc := github.NewClient(c)

		err := gh.NewIssueHelper(gc, "Markup", ghRepoName).Comment(context.Background(), issue, commentBody)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create comment")
	})

	t.Run("working as expected", func(t *testing.T) {
		c := mock.NewMockedHTTPClient(
			mock.WithRequestMatchHandler(
				mock.PostReposIssuesCommentsByOwnerByRepoByIssueNumber,
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					m := make(map[string]interface{})
					assert.NoError(
						t,
						json.NewDecoder(r.Body).Decode(&m),
					)
					assert.Equal(t, commentBody, m["body"])
					w.WriteHeader(http.StatusCreated)
					responseBody := commentBody
					assert.NoError(
						t,
						json.NewEncoder(w).Encode(&github.IssueComment{Body: &responseBody}),
					)
				}),
			),
		)

		gc := github.NewClient(c)

		err := gh.NewIssueHelper(gc, "Markup", ghRepoName).Comment(context.Background(), issue, commentBody)

		assert.NoError(t, err)
	})
}
