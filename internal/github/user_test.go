package github_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"path"
	"testing"

	"github.com/google/go-github/v47/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	gh "github.com/qbarrand/gitstream/internal/github"
	"github.com/stretchr/testify/assert"
)

func TestUserHelperImpl_GetCommitAuthor(t *testing.T) {

	const (
		owner     = "some-owner"
		repo      = "some-repo"
		commitSha = "some-sha"
	)

	var (
		authorLogin = "suser"
	)

	ctx := context.Background()

	q := url.Values{}
	q.Add("hash", commitSha)
	q.Add("repo", path.Join(owner, repo))
	expectedQuery := q.Encode()

	t.Run("Github API error", func(t *testing.T) {

		c := mock.NewMockedHTTPClient(
			mock.WithRequestMatchHandler(
				mock.GetSearchCommits,
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, expectedQuery, r.URL.Query().Get("q"))
					w.WriteHeader(http.StatusServiceUnavailable)
				}),
			),
		)

		gc := github.NewClient(c)

		_, err := gh.NewUserHelper(gc, &gh.RepoName{Owner: owner, Repo: repo}).GetCommitAuthor(ctx, commitSha)

		assert.Error(t, err)
		assert.ErrorContains(t, err, "failed to get commit")
	})

	t.Run("commit not found", func(t *testing.T) {

		commitsCount := 0
		commitsSearchRes := &github.CommitsSearchResult{
			Total:   &commitsCount,
			Commits: []*github.CommitResult{},
		}

		c := mock.NewMockedHTTPClient(
			mock.WithRequestMatchHandler(
				mock.GetSearchCommits,
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, expectedQuery, r.URL.Query().Get("q"))
					assert.NoError(
						t,
						json.NewEncoder(w).Encode(commitsSearchRes),
					)
				}),
			),
		)

		gc := github.NewClient(c)

		_, err := gh.NewUserHelper(gc, &gh.RepoName{Owner: owner, Repo: repo}).GetCommitAuthor(ctx, commitSha)

		assert.Error(t, err)
		assert.ErrorContains(t, err, "there are 0 commits matching the search query")
		assert.ErrorIs(t, err, gh.ErrUnexpectedReply)
	})

	t.Run("more than 1 commit found", func(t *testing.T) {

		commitsCount := 2
		otherUserLogin := "some other user"
		commitsSearchRes := &github.CommitsSearchResult{
			Total: &commitsCount,
			Commits: []*github.CommitResult{
				{
					Author: &github.User{
						Login: &authorLogin,
					},
				},
				{
					Author: &github.User{
						Login: &otherUserLogin,
					},
				},
			},
		}

		c := mock.NewMockedHTTPClient(
			mock.WithRequestMatchHandler(
				mock.GetSearchCommits,
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, expectedQuery, r.URL.Query().Get("q"))
					assert.NoError(
						t,
						json.NewEncoder(w).Encode(commitsSearchRes),
					)
				}),
			),
		)

		gc := github.NewClient(c)

		_, err := gh.NewUserHelper(gc, &gh.RepoName{Owner: owner, Repo: repo}).GetCommitAuthor(ctx, commitSha)

		assert.Error(t, err)
		assert.ErrorContains(t, err, "there are 2 commits matching the search query")
		assert.ErrorIs(t, err, gh.ErrUnexpectedReply)
	})

	t.Run("working as expected", func(t *testing.T) {

		commitsCount := 1
		commitsSearchRes := &github.CommitsSearchResult{
			Total: &commitsCount,
			Commits: []*github.CommitResult{
				{
					Author: &github.User{
						Login: &authorLogin,
					},
				},
			},
		}

		c := mock.NewMockedHTTPClient(
			mock.WithRequestMatchHandler(
				mock.GetSearchCommits,
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, expectedQuery, r.URL.Query().Get("q"))
					assert.NoError(
						t,
						json.NewEncoder(w).Encode(commitsSearchRes),
					)
				}),
			),
		)

		gc := github.NewClient(c)

		user, err := gh.NewUserHelper(gc, &gh.RepoName{Owner: owner, Repo: repo}).GetCommitAuthor(ctx, commitSha)

		assert.NoError(t, err)
		assert.Equal(t, *user.Login, authorLogin)
	})
}
