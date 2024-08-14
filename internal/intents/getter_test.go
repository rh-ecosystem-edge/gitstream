package intents_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	"github.com/google/go-github/v47/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	gh "github.com/rh-ecosystem-edge/gitstream/internal/github"
	"github.com/rh-ecosystem-edge/gitstream/internal/intents"
	"github.com/rh-ecosystem-edge/gitstream/internal/markup"
	"github.com/stretchr/testify/assert"
)

func TestNewIntentsGetter(t *testing.T) {
	ig := intents.NewIntentsGetter(nil, nil, logr.Discard())

	assert.NotNil(t, ig)
}

func TestGetterImpl_FromGitHubIssues(t *testing.T) {
	t.Run("GitHub returns an error", func(t *testing.T) {
		mockedHTTPClient := mock.NewMockedHTTPClient(
			mock.WithRequestMatchHandler(
				mock.GetReposIssuesByOwnerByRepo,
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					mock.WriteError(w, http.StatusInternalServerError, "there is some issue")
				}),
			),
		)

		c := github.NewClient(mockedHTTPClient)
		ig := intents.NewIntentsGetter(nil, c, logr.Discard())

		_, err := ig.FromGitHubIssues(context.Background(), &gh.RepoName{})
		assert.Error(t, err)
	})

	t.Run("GitHub returns one qualifying issue", func(t *testing.T) {
		const (
			hashStr   = "e3229f3c533ed51070beff092e5c7694a8ee81f0"
			issueURL0 = "some-issue-url-0"
			issueURL1 = "some-issue-url-1"
			msg0      = "Message 0"
			msg1      = "Message 1"
		)

		repoName := gh.RepoName{Owner: "owner", Repo: "repo"}

		mockedHTTPClient := mock.NewMockedHTTPClient(
			mock.WithRequestMatch(
				mock.GetReposIssuesByOwnerByRepo,
				[]github.Issue{
					{
						Body:    github.String(msg0),
						HTMLURL: github.String(issueURL0),
					},
					{
						Body:    github.String(msg1),
						HTMLURL: github.String(issueURL1),
					},
				},
				[]github.Issue{
					{
						HTMLURL: github.String("some-url"),
					},
				},
			),
		)

		c := github.NewClient(mockedHTTPClient)

		ctrl := gomock.NewController(t)

		finder := markup.NewMockFinder(ctrl)

		hash := plumbing.NewHash(hashStr)

		gomock.InOrder(
			finder.EXPECT().FindSHAs(msg0),
			finder.EXPECT().FindSHAs(msg1).Return([]plumbing.Hash{hash}, nil),
		)

		ci, err := intents.NewIntentsGetter(finder, c, logr.Discard()).FromGitHubIssues(context.Background(), &repoName)
		assert.NoError(t, err)
		assert.Equal(t, intents.CommitIntents{hash: issueURL1}, ci)

	})
}

func TestGetterImpl_FromLocalGitRepo(t *testing.T) {
	// TODO
}

func TestMergeCommitIntents(t *testing.T) {
	hash1 := plumbing.NewHash("e3229f3c533ed51070beff092e5c7694a8ee81f0")
	hash2 := plumbing.NewHash("9c08d42326af62aa0f8cea021c4d37971606148f")

	t.Run("should combine commit intents", func(t *testing.T) {
		m := intents.MergeCommitIntents(
			intents.CommitIntents{hash1: "origin 0"},
			intents.CommitIntents{hash2: "origin 2"},
		)

		assert.Len(t, m, 2)
		assert.Contains(t, m, hash1)
		assert.Contains(t, m, hash2)
	})

	t.Run("double override", func(t *testing.T) {
		final := intents.CommitIntents{hash1: "origin 2"}

		m := intents.MergeCommitIntents(
			intents.CommitIntents{hash1: "origin 0"},
			intents.CommitIntents{hash1: "origin 1"},
			final,
		)

		assert.Equal(t, final, m)
	})
}
