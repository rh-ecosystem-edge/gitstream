package gitstream

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	"github.com/google/go-github/v47/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	"github.com/qbarrand/gitstream/internal/config"
	gh "github.com/qbarrand/gitstream/internal/github"
	"github.com/qbarrand/gitstream/internal/gitutils"
	"github.com/qbarrand/gitstream/internal/markup"
	"github.com/qbarrand/gitstream/internal/test"
	"github.com/stretchr/testify/assert"
)

func TestAssign_getOwnersContent(t *testing.T) {

	const (
		repoOwner          = "owner"
		repoName           = "repo"
		upstreamMainBranch = "us-main"
		upstreamURL        = "some-upstream-url"
	)

	t.Run("github API error", func(t *testing.T) {

		ctx := context.Background()

		c := mock.NewMockedHTTPClient(
			mock.WithRequestMatchHandler(
				mock.GetReposContentsByOwnerByRepoByPath,
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Contains(t, r.URL.Path, "OWNERS")
					w.WriteHeader(http.StatusBadRequest)
				}),
			),
		)

		gc := github.NewClient(c)
		ctrl := gomock.NewController(t)

		mockFinder := markup.NewMockFinder(ctrl)
		mockGitHelper := gitutils.NewMockHelper(ctrl)
		mockIssueHelper := gh.NewMockIssueHelper(ctrl)
		mockUserHelper := gh.NewMockUserHelper(ctrl)

		repo, _ := test.CloneCurrentRepoWithFS(t)
		ghRepoName := &gh.RepoName{
			Owner: repoOwner,
			Repo:  repoName,
		}

		upstreamConfig := config.Upstream{
			Ref: upstreamMainBranch,
			URL: upstreamURL,
		}

		a := Assign{
			GC:             gc,
			DryRun:         false,
			Finder:         mockFinder,
			GitHelper:      mockGitHelper,
			Logger:         logr.Discard(),
			IssueHelper:    mockIssueHelper,
			UserHelper:     mockUserHelper,
			Repo:           repo,
			RepoName:       ghRepoName,
			UpstreamConfig: upstreamConfig,
		}

		_, err := a.getOwnersContent(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "could not get OWNERS's content from")
	})

	t.Run("failed to unmarshal OWNERS file", func(t *testing.T) {

		var (
			ctx      = context.Background()
			encoding = "base64"
			content  = "bm9uIHZhbGlkIGRhdGEK" // base64("non valid data")
		)

		expectedContent := &github.RepositoryContent{
			Encoding: &encoding,
			Content:  &content,
		}

		c := mock.NewMockedHTTPClient(
			mock.WithRequestMatchHandler(
				mock.GetReposContentsByOwnerByRepoByPath,
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Contains(t, r.URL.Path, "OWNERS")
					assert.NoError(
						t,
						json.NewEncoder(w).Encode(expectedContent),
					)
				}),
			),
		)

		gc := github.NewClient(c)
		ctrl := gomock.NewController(t)

		mockFinder := markup.NewMockFinder(ctrl)
		mockGitHelper := gitutils.NewMockHelper(ctrl)
		mockIssueHelper := gh.NewMockIssueHelper(ctrl)
		mockUserHelper := gh.NewMockUserHelper(ctrl)

		repo, _ := test.CloneCurrentRepoWithFS(t)
		ghRepoName := &gh.RepoName{
			Owner: repoOwner,
			Repo:  repoName,
		}

		upstreamConfig := config.Upstream{
			Ref: upstreamMainBranch,
			URL: upstreamURL,
		}

		a := Assign{
			GC:             gc,
			DryRun:         false,
			Finder:         mockFinder,
			GitHelper:      mockGitHelper,
			Logger:         logr.Discard(),
			IssueHelper:    mockIssueHelper,
			UserHelper:     mockUserHelper,
			Repo:           repo,
			RepoName:       ghRepoName,
			UpstreamConfig: upstreamConfig,
		}

		_, err := a.getOwnersContent(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "could not unmarshal")
	})

	t.Run("working as expected", func(t *testing.T) {

		const content = `
approvers:
  - user1
reviewers:
  - user1
  - user2
`
		var (
			ctx            = context.Background()
			encoding       = "base64"
			encodedContent = base64.StdEncoding.EncodeToString([]byte(content))
		)

		expectedContent := &github.RepositoryContent{
			Encoding: &encoding,
			Content:  &encodedContent,
		}

		c := mock.NewMockedHTTPClient(
			mock.WithRequestMatchHandler(
				mock.GetReposContentsByOwnerByRepoByPath,
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Contains(t, r.URL.Path, "OWNERS")
					assert.NoError(
						t,
						json.NewEncoder(w).Encode(expectedContent),
					)
				}),
			),
		)

		gc := github.NewClient(c)
		ctrl := gomock.NewController(t)

		mockFinder := markup.NewMockFinder(ctrl)
		mockGitHelper := gitutils.NewMockHelper(ctrl)
		mockIssueHelper := gh.NewMockIssueHelper(ctrl)
		mockUserHelper := gh.NewMockUserHelper(ctrl)

		repo, _ := test.CloneCurrentRepoWithFS(t)
		ghRepoName := &gh.RepoName{
			Owner: repoOwner,
			Repo:  repoName,
		}

		upstreamConfig := config.Upstream{
			Ref: upstreamMainBranch,
			URL: upstreamURL,
		}

		a := Assign{
			GC:             gc,
			DryRun:         false,
			Finder:         mockFinder,
			GitHelper:      mockGitHelper,
			Logger:         logr.Discard(),
			IssueHelper:    mockIssueHelper,
			UserHelper:     mockUserHelper,
			Repo:           repo,
			RepoName:       ghRepoName,
			UpstreamConfig: upstreamConfig,
		}

		owners, err := a.getOwnersContent(ctx)
		assert.NoError(t, err)
		assert.True(t, contains(owners.Approvers, "user1"))
	})
}

func contains(slice []string, str string) bool {

	for _, s := range slice {
		if s == str {
			return true
		}
	}

	return false
}

func TestAssign_assignIssues(t *testing.T) {

	const (
		repoOwner          = "owner"
		repoName           = "repo"
		upstreamMainBranch = "us-main"
		upstreamURL        = "some-upstream-url"
	)

	const content = `
approvers:
  - user1
reviewers:
  - user1
  - user2
`

	var (
		encoding       = "base64"
		encodedContent = base64.StdEncoding.EncodeToString([]byte(content))
	)

	t.Run("failed to list open issues", func(t *testing.T) {

		var (
			ctx = context.Background()
		)

		expectedContent := &github.RepositoryContent{
			Encoding: &encoding,
			Content:  &encodedContent,
		}

		c := mock.NewMockedHTTPClient(
			mock.WithRequestMatchHandler(
				mock.GetReposContentsByOwnerByRepoByPath,
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Contains(t, r.URL.Path, "OWNERS")
					assert.NoError(
						t,
						json.NewEncoder(w).Encode(expectedContent),
					)
				}),
			),
		)

		gc := github.NewClient(c)
		ctrl := gomock.NewController(t)

		mockFinder := markup.NewMockFinder(ctrl)
		mockGitHelper := gitutils.NewMockHelper(ctrl)
		mockIssueHelper := gh.NewMockIssueHelper(ctrl)
		mockUserHelper := gh.NewMockUserHelper(ctrl)

		repo, _ := test.CloneCurrentRepoWithFS(t)
		ghRepoName := &gh.RepoName{
			Owner: repoOwner,
			Repo:  repoName,
		}

		upstreamConfig := config.Upstream{
			Ref: upstreamMainBranch,
			URL: upstreamURL,
		}

		a := Assign{
			GC:             gc,
			DryRun:         false,
			Finder:         mockFinder,
			GitHelper:      mockGitHelper,
			Logger:         logr.Discard(),
			IssueHelper:    mockIssueHelper,
			UserHelper:     mockUserHelper,
			Repo:           repo,
			RepoName:       ghRepoName,
			UpstreamConfig: upstreamConfig,
		}

		gomock.InOrder(
			mockIssueHelper.EXPECT().ListAllOpen(ctx, true).Return([]*github.Issue{}, errors.New("some error")),
		)

		err := a.assignIssues(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "could not list open issues")
	})

	t.Run("do nothing if the issue is already assigned", func(t *testing.T) {

		var (
			ctx = context.Background()
		)

		expectedContent := &github.RepositoryContent{
			Encoding: &encoding,
			Content:  &encodedContent,
		}

		c := mock.NewMockedHTTPClient(
			mock.WithRequestMatchHandler(
				mock.GetReposContentsByOwnerByRepoByPath,
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Contains(t, r.URL.Path, "OWNERS")
					assert.NoError(
						t,
						json.NewEncoder(w).Encode(expectedContent),
					)
				}),
			),
		)

		gc := github.NewClient(c)
		ctrl := gomock.NewController(t)

		mockFinder := markup.NewMockFinder(ctrl)
		mockGitHelper := gitutils.NewMockHelper(ctrl)
		mockIssueHelper := gh.NewMockIssueHelper(ctrl)
		mockUserHelper := gh.NewMockUserHelper(ctrl)

		repo, _ := test.CloneCurrentRepoWithFS(t)
		ghRepoName := &gh.RepoName{
			Owner: repoOwner,
			Repo:  repoName,
		}

		upstreamConfig := config.Upstream{
			Ref: upstreamMainBranch,
			URL: upstreamURL,
		}

		a := Assign{
			GC:             gc,
			DryRun:         false,
			Finder:         mockFinder,
			GitHelper:      mockGitHelper,
			Logger:         logr.Discard(),
			IssueHelper:    mockIssueHelper,
			UserHelper:     mockUserHelper,
			Repo:           repo,
			RepoName:       ghRepoName,
			UpstreamConfig: upstreamConfig,
		}

		var (
			issueNumber = 123
			issueURL    = "some url"
			body        = "some body"
			login       = "user1"
		)
		issues := []*github.Issue{
			{
				Number:  &issueNumber,
				HTMLURL: &issueURL,
				Body:    &body,
				Assignees: []*github.User{
					{
						Login: &login,
					},
				},
			},
		}
		gomock.InOrder(
			mockIssueHelper.EXPECT().ListAllOpen(ctx, true).Return(issues, nil),
		)

		err := a.assignIssues(ctx)
		assert.NoError(t, err)
	})

	t.Run("failed to find SHAs", func(t *testing.T) {

		var (
			ctx = context.Background()
		)

		expectedContent := &github.RepositoryContent{
			Encoding: &encoding,
			Content:  &encodedContent,
		}

		c := mock.NewMockedHTTPClient(
			mock.WithRequestMatchHandler(
				mock.GetReposContentsByOwnerByRepoByPath,
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Contains(t, r.URL.Path, "OWNERS")
					assert.NoError(
						t,
						json.NewEncoder(w).Encode(expectedContent),
					)
				}),
			),
		)

		gc := github.NewClient(c)
		ctrl := gomock.NewController(t)

		mockFinder := markup.NewMockFinder(ctrl)
		mockGitHelper := gitutils.NewMockHelper(ctrl)
		mockIssueHelper := gh.NewMockIssueHelper(ctrl)
		mockUserHelper := gh.NewMockUserHelper(ctrl)

		repo, _ := test.CloneCurrentRepoWithFS(t)
		ghRepoName := &gh.RepoName{
			Owner: repoOwner,
			Repo:  repoName,
		}

		upstreamConfig := config.Upstream{
			Ref: upstreamMainBranch,
			URL: upstreamURL,
		}

		a := Assign{
			GC:             gc,
			DryRun:         false,
			Finder:         mockFinder,
			GitHelper:      mockGitHelper,
			Logger:         logr.Discard(),
			IssueHelper:    mockIssueHelper,
			UserHelper:     mockUserHelper,
			Repo:           repo,
			RepoName:       ghRepoName,
			UpstreamConfig: upstreamConfig,
		}

		var (
			issueNumber = 123
			issueURL    = "some url"
			body        = "some body"
		)
		issues := []*github.Issue{
			{
				Number:  &issueNumber,
				HTMLURL: &issueURL,
				Body:    &body,
			},
		}

		gomock.InOrder(
			mockIssueHelper.EXPECT().ListAllOpen(ctx, true).Return(issues, nil),
			mockFinder.EXPECT().FindSHAs(gomock.Any()).Return([]plumbing.Hash{}, errors.New("some error")),
		)

		err := a.assignIssues(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error while looking for SHAs")
	})

	t.Run("failed to get user", func(t *testing.T) {

		var (
			ctx = context.Background()
		)

		expectedContent := &github.RepositoryContent{
			Encoding: &encoding,
			Content:  &encodedContent,
		}

		c := mock.NewMockedHTTPClient(
			mock.WithRequestMatchHandler(
				mock.GetReposContentsByOwnerByRepoByPath,
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Contains(t, r.URL.Path, "OWNERS")
					assert.NoError(
						t,
						json.NewEncoder(w).Encode(expectedContent),
					)
				}),
			),
		)

		gc := github.NewClient(c)
		ctrl := gomock.NewController(t)

		mockFinder := markup.NewMockFinder(ctrl)
		mockGitHelper := gitutils.NewMockHelper(ctrl)
		mockIssueHelper := gh.NewMockIssueHelper(ctrl)
		mockUserHelper := gh.NewMockUserHelper(ctrl)

		repo, _ := test.CloneCurrentRepoWithFS(t)
		ghRepoName := &gh.RepoName{
			Owner: repoOwner,
			Repo:  repoName,
		}

		upstreamConfig := config.Upstream{
			Ref: upstreamMainBranch,
			URL: upstreamURL,
		}

		a := Assign{
			GC:             gc,
			DryRun:         false,
			Finder:         mockFinder,
			GitHelper:      mockGitHelper,
			Logger:         logr.Discard(),
			IssueHelper:    mockIssueHelper,
			UserHelper:     mockUserHelper,
			Repo:           repo,
			RepoName:       ghRepoName,
			UpstreamConfig: upstreamConfig,
		}

		var (
			issueNumber = 123
			issueURL    = "some url"
			body        = "some body"
		)
		issues := []*github.Issue{
			{
				Number:  &issueNumber,
				HTMLURL: &issueURL,
				Body:    &body,
			},
		}

		ref, err := repo.Head()
		assert.NoError(t, err)
		hashes := []plumbing.Hash{ref.Hash()}

		gomock.InOrder(
			mockIssueHelper.EXPECT().ListAllOpen(ctx, true).Return(issues, nil),
			mockFinder.EXPECT().FindSHAs(gomock.Any()).Return(hashes, nil),
			mockUserHelper.EXPECT().GetUser(ctx, gomock.Any()).Return(nil, errors.New("some error")),
			mockIssueHelper.EXPECT().Assign(ctx, issues[0], gomock.Any()).Return(nil),
		)

		err = a.assignIssues(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "could not get the upstream commit author for downstream issue")
	})

	t.Run("failed to assign an issue", func(t *testing.T) {

		var (
			ctx = context.Background()
		)

		expectedContent := &github.RepositoryContent{
			Encoding: &encoding,
			Content:  &encodedContent,
		}

		c := mock.NewMockedHTTPClient(
			mock.WithRequestMatchHandler(
				mock.GetReposContentsByOwnerByRepoByPath,
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Contains(t, r.URL.Path, "OWNERS")
					assert.NoError(
						t,
						json.NewEncoder(w).Encode(expectedContent),
					)
				}),
			),
		)

		gc := github.NewClient(c)
		ctrl := gomock.NewController(t)

		mockFinder := markup.NewMockFinder(ctrl)
		mockGitHelper := gitutils.NewMockHelper(ctrl)
		mockIssueHelper := gh.NewMockIssueHelper(ctrl)
		mockUserHelper := gh.NewMockUserHelper(ctrl)

		repo, _ := test.CloneCurrentRepoWithFS(t)
		ghRepoName := &gh.RepoName{
			Owner: repoOwner,
			Repo:  repoName,
		}

		upstreamConfig := config.Upstream{
			Ref: upstreamMainBranch,
			URL: upstreamURL,
		}

		a := Assign{
			GC:             gc,
			DryRun:         false,
			Finder:         mockFinder,
			GitHelper:      mockGitHelper,
			Logger:         logr.Discard(),
			IssueHelper:    mockIssueHelper,
			UserHelper:     mockUserHelper,
			Repo:           repo,
			RepoName:       ghRepoName,
			UpstreamConfig: upstreamConfig,
		}

		var (
			issueNumber = 123
			issueURL    = "some url"
			body        = "some body"
			userLogin   = "user1"
		)
		issues := []*github.Issue{
			{
				Number:  &issueNumber,
				HTMLURL: &issueURL,
				Body:    &body,
			},
		}

		ref, err := repo.Head()
		assert.NoError(t, err)
		hashes := []plumbing.Hash{ref.Hash()}

		user := &github.User{
			Login: &userLogin,
		}

		gomock.InOrder(
			mockIssueHelper.EXPECT().ListAllOpen(ctx, true).Return(issues, nil),
			mockFinder.EXPECT().FindSHAs(gomock.Any()).Return(hashes, nil),
			mockUserHelper.EXPECT().GetUser(ctx, gomock.Any()).Return(user, nil),
			mockIssueHelper.EXPECT().Assign(ctx, issues[0], userLogin).Return(errors.New("error")),
		)

		err = a.assignIssues(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "could not assign issue")
	})

	t.Run("working as expected", func(t *testing.T) {

		var (
			ctx = context.Background()
		)

		expectedContent := &github.RepositoryContent{
			Encoding: &encoding,
			Content:  &encodedContent,
		}

		c := mock.NewMockedHTTPClient(
			mock.WithRequestMatchHandler(
				mock.GetReposContentsByOwnerByRepoByPath,
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Contains(t, r.URL.Path, "OWNERS")
					assert.NoError(
						t,
						json.NewEncoder(w).Encode(expectedContent),
					)
				}),
			),
		)

		gc := github.NewClient(c)
		ctrl := gomock.NewController(t)

		mockFinder := markup.NewMockFinder(ctrl)
		mockGitHelper := gitutils.NewMockHelper(ctrl)
		mockIssueHelper := gh.NewMockIssueHelper(ctrl)
		mockUserHelper := gh.NewMockUserHelper(ctrl)

		repo, _ := test.CloneCurrentRepoWithFS(t)
		ghRepoName := &gh.RepoName{
			Owner: repoOwner,
			Repo:  repoName,
		}

		upstreamConfig := config.Upstream{
			Ref: upstreamMainBranch,
			URL: upstreamURL,
		}

		a := Assign{
			GC:             gc,
			DryRun:         false,
			Finder:         mockFinder,
			GitHelper:      mockGitHelper,
			Logger:         logr.Discard(),
			IssueHelper:    mockIssueHelper,
			UserHelper:     mockUserHelper,
			Repo:           repo,
			RepoName:       ghRepoName,
			UpstreamConfig: upstreamConfig,
		}

		var (
			issueNumber = 123
			issueURL    = "some url"
			body        = "some body"
			userLogin   = "user1"
		)
		issues := []*github.Issue{
			{
				Number:  &issueNumber,
				HTMLURL: &issueURL,
				Body:    &body,
			},
		}

		ref, err := repo.Head()
		assert.NoError(t, err)
		hashes := []plumbing.Hash{ref.Hash()}

		user := &github.User{
			Login: &userLogin,
		}

		gomock.InOrder(
			mockIssueHelper.EXPECT().ListAllOpen(ctx, true).Return(issues, nil),
			mockFinder.EXPECT().FindSHAs(gomock.Any()).Return(hashes, nil),
			mockUserHelper.EXPECT().GetUser(ctx, gomock.Any()).Return(user, nil),
			mockIssueHelper.EXPECT().Assign(ctx, issues[0], userLogin).Return(nil),
		)

		err = a.assignIssues(ctx)
		assert.NoError(t, err)
	})
}
