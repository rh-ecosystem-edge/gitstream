package github_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/go-github/v47/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	gh "github.com/qbarrand/gitstream/internal/github"
	"github.com/stretchr/testify/assert"
)

func TestUserHelperImpl_GetUser(t *testing.T) {

	const (
		owner       = "owner"
		repo        = "repo"
		authorEmail = "suser@redhat.com"
	)

	var (
		authorName  = "Some User"
		authorLogin = "suser"
	)

	ctx := context.Background()

	commit := &object.Commit{
		Author: object.Signature{
			Name:  authorName,
			Email: authorEmail,
		},
	}

	t.Run("could not get github user", func(t *testing.T) {

		c := mock.NewMockedHTTPClient(
			mock.WithRequestMatchHandler(
				mock.GetSearchUsers,
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, authorEmail, r.URL.Query().Get("q"))
					w.WriteHeader(http.StatusBadRequest)
				}),
			),
		)

		gc := github.NewClient(c)

		_, err := gh.NewUserHelper(gc, &gh.RepoName{Owner: owner, Repo: repo}).GetUser(ctx, commit)

		assert.Error(t, err)
		assert.ErrorContains(t, err, "could not get user from email")
	})

	t.Run("there is more than 1 user associated with the commit author eamil", func(t *testing.T) {

		userSearchRes := &github.UsersSearchResult{
			Users: []*github.User{
				{
					Login: &authorLogin,
				},
				{
					Login: &authorLogin,
				},
			},
		}

		c := mock.NewMockedHTTPClient(
			mock.WithRequestMatchHandler(
				mock.GetSearchUsers,
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, authorEmail, r.URL.Query().Get("q"))
					assert.NoError(
						t,
						json.NewEncoder(w).Encode(userSearchRes),
					)
				}),
			),
		)

		gc := github.NewClient(c)

		_, err := gh.NewUserHelper(gc, &gh.RepoName{Owner: owner, Repo: repo}).GetUser(ctx, commit)

		assert.Error(t, err)
		assert.ErrorContains(t, err, "there is more than 1 user associated with")
	})

	t.Run("working as expected", func(t *testing.T) {

		userSearchRes := &github.UsersSearchResult{
			Users: []*github.User{
				{
					Login: &authorLogin,
				},
			},
		}

		c := mock.NewMockedHTTPClient(
			mock.WithRequestMatchHandler(
				mock.GetSearchUsers,
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, authorEmail, r.URL.Query().Get("q"))
					assert.NoError(
						t,
						json.NewEncoder(w).Encode(userSearchRes),
					)
				}),
			),
		)

		gc := github.NewClient(c)

		user, err := gh.NewUserHelper(gc, &gh.RepoName{Owner: owner, Repo: repo}).GetUser(ctx, commit)

		assert.NoError(t, err)
		assert.Equal(t, *user.Login, authorLogin)
	})
}
