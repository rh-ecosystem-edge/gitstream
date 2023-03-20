package gitstream

import (
	"context"
	"errors"
	"testing"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	"github.com/google/go-github/v47/github"
	"github.com/qbarrand/gitstream/internal/config"
	gh "github.com/qbarrand/gitstream/internal/github"
	"github.com/qbarrand/gitstream/internal/gitutils"
	"github.com/qbarrand/gitstream/internal/markup"
	"github.com/qbarrand/gitstream/internal/owners"
	"github.com/qbarrand/gitstream/internal/test"
	"github.com/stretchr/testify/assert"
)

func TestAssign_filterApproversFromCommitAuthors(t *testing.T) {

	const (
		upstreamMainBranch = "us-main"
		upstreamURL        = "some-upstream-url"
		ownersFileName     = "OWNERS"
	)

	ctrl := gomock.NewController(t)
	mockOwnersHelper := owners.NewMockOwnersHelper(ctrl)

	upstreamConfig := config.Upstream{
		Ref: upstreamMainBranch,
		URL: upstreamURL,
	}

	downstreamConfig := config.Downstream{
		OwnersFile: ownersFileName,
	}

	a := Assign{
		Finder:           nil,
		GitHelper:        nil,
		Logger:           logr.Discard(),
		IssueHelper:      nil,
		UserHelper:       nil,
		Repo:             test.NewRepo(t),
		RepoName:         nil,
		UpstreamConfig:   upstreamConfig,
		DownstreamConfig: downstreamConfig,
		OwnersHelper:     mockOwnersHelper,
	}

	t.Run("working as expected", func(t *testing.T) {

		commitAuthors := []string{
			"user1",
			"approver1",
			"user2",
			"approver2",
		}

		o := &owners.Owners{
			Approvers: []string{
				"approver1",
				"approver2",
			},
		}

		expectedFilteredCommitAuthors := []string{
			"approver1",
			"approver2",
		}

		gomock.InOrder(
			mockOwnersHelper.EXPECT().IsApprover(o, "user1").Return(false),
			mockOwnersHelper.EXPECT().IsApprover(o, "approver1").Return(true),
			mockOwnersHelper.EXPECT().IsApprover(o, "user2").Return(false),
			mockOwnersHelper.EXPECT().IsApprover(o, "approver2").Return(true),
		)

		filteredCommitAuthors := a.filterApproversFromCommitAuthors(commitAuthors, o)

		assert.Equal(t, filteredCommitAuthors, expectedFilteredCommitAuthors)
	})
}

func TestAssign_handleIssue(t *testing.T) {

	var (
		ctx = context.Background()
	)

	o := &owners.Owners{
		Approvers: []string{
			"approver",
		},
	}

	t.Run("nothing should happen if the issue is already assigned", func(t *testing.T) {

		var (
			issueNumber = 123
			issueURL    = "some url"
			login       = "user1"
		)

		a := Assign{
			Logger: logr.Discard(),
		}

		issue := &github.Issue{
			Number:  &issueNumber,
			HTMLURL: &issueURL,
			Assignees: []*github.User{
				{
					Login: &login,
				},
			},
		}

		err := a.handleIssue(ctx, issue, o)
		assert.NoError(t, err)
	})

	t.Run("failed to find SHAs", func(t *testing.T) {

		var (
			ctx         = context.Background()
			issueNumber = 123
			issueURL    = "some url"
			body        = "some body"
		)

		ctrl := gomock.NewController(t)

		mockFinder := markup.NewMockFinder(ctrl)

		a := Assign{
			Finder: mockFinder,
			Logger: logr.Discard(),
		}

		issue := &github.Issue{
			Number:  &issueNumber,
			HTMLURL: &issueURL,
			Body:    &body,
		}

		gomock.InOrder(
			mockFinder.EXPECT().FindSHAs(gomock.Any()).Return([]plumbing.Hash{}, errors.New("some error")),
		)

		err := a.handleIssue(ctx, issue, o)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error while looking for SHAs")
	})

	t.Run("failed to get commit author", func(t *testing.T) {

		var (
			ctx         = context.Background()
			issueNumber = 123
			issueURL    = "some url"
			body        = "some body"
		)

		ctrl := gomock.NewController(t)

		mockFinder := markup.NewMockFinder(ctrl)
		mockUserHelper := gh.NewMockUserHelper(ctrl)
		mockIssueHelper := gh.NewMockIssueHelper(ctrl)

		a := Assign{
			Finder:      mockFinder,
			Logger:      logr.Discard(),
			UserHelper:  mockUserHelper,
			IssueHelper: mockIssueHelper,
		}

		issue := &github.Issue{
			Number:  &issueNumber,
			HTMLURL: &issueURL,
			Body:    &body,
		}

		repo := test.NewRepo(t)
		sha, _ := test.AddEmptyCommit(t, repo, "empty")

		gomock.InOrder(
			mockFinder.EXPECT().FindSHAs(body).Return([]plumbing.Hash{sha}, nil),
			mockUserHelper.EXPECT().GetCommitAuthor(ctx, sha.String()).Return(nil, errors.New("some API error")),
		)

		err := a.handleIssue(ctx, issue, o)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get commit author from GitHub for issue")
	})

	t.Run("user is NOT an approver, we failed to get a random approver", func(t *testing.T) {

		var (
			ctx         = context.Background()
			issueNumber = 123
			issueURL    = "some url"
			body        = "some body"
			nonApprover = "notanapprover"
		)

		ctrl := gomock.NewController(t)

		mockFinder := markup.NewMockFinder(ctrl)
		mockUserHelper := gh.NewMockUserHelper(ctrl)
		mockOwnersHelper := owners.NewMockOwnersHelper(ctrl)

		a := Assign{
			Finder:       mockFinder,
			Logger:       logr.Discard(),
			OwnersHelper: mockOwnersHelper,
			UserHelper:   mockUserHelper,
		}

		issue := &github.Issue{
			Number:  &issueNumber,
			HTMLURL: &issueURL,
			Body:    &body,
		}

		repo := test.NewRepo(t)
		sha, _ := test.AddEmptyCommit(t, repo, "empty")
		user := &github.User{
			Login: &nonApprover,
		}

		gomock.InOrder(
			mockFinder.EXPECT().FindSHAs(body).Return([]plumbing.Hash{sha}, nil),
			mockUserHelper.EXPECT().GetCommitAuthor(ctx, sha.String()).Return(user, nil),
			mockOwnersHelper.EXPECT().IsApprover(o, *user.Login).Return(false),
			mockOwnersHelper.EXPECT().GetRandomApprover(o).Return("", errors.New("some error")),
		)

		err := a.handleIssue(ctx, issue, o)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "could not get a random approver")
	})

	t.Run("user is an approver, failed to assign issue", func(t *testing.T) {

		var (
			ctx         = context.Background()
			issueNumber = 123
			issueURL    = "some url"
			body        = "some body"
		)

		ctrl := gomock.NewController(t)

		mockFinder := markup.NewMockFinder(ctrl)
		mockUserHelper := gh.NewMockUserHelper(ctrl)
		mockOwnersHelper := owners.NewMockOwnersHelper(ctrl)
		mockIssueHelper := gh.NewMockIssueHelper(ctrl)

		a := Assign{
			Finder:       mockFinder,
			Logger:       logr.Discard(),
			OwnersHelper: mockOwnersHelper,
			UserHelper:   mockUserHelper,
			IssueHelper:  mockIssueHelper,
		}

		issue := &github.Issue{
			Number:  &issueNumber,
			HTMLURL: &issueURL,
			Body:    &body,
		}

		repo := test.NewRepo(t)
		sha, _ := test.AddEmptyCommit(t, repo, "empty")
		user := &github.User{
			Login: &o.Approvers[0],
		}

		gomock.InOrder(
			mockFinder.EXPECT().FindSHAs(body).Return([]plumbing.Hash{sha}, nil),
			mockUserHelper.EXPECT().GetCommitAuthor(ctx, sha.String()).Return(user, nil),
			mockOwnersHelper.EXPECT().IsApprover(o, *user.Login).Return(true),
			mockIssueHelper.EXPECT().Assign(ctx, issue, *user.Login).Return(errors.New("error")),
		)

		err := a.handleIssue(ctx, issue, o)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "could not assign issue")
	})

	t.Run("working as expected, use is an approver", func(t *testing.T) {

		var (
			ctx         = context.Background()
			issueNumber = 123
			issueURL    = "some url"
			body        = "some body"
		)

		ctrl := gomock.NewController(t)

		mockFinder := markup.NewMockFinder(ctrl)
		mockUserHelper := gh.NewMockUserHelper(ctrl)
		mockOwnersHelper := owners.NewMockOwnersHelper(ctrl)
		mockIssueHelper := gh.NewMockIssueHelper(ctrl)

		a := Assign{
			Finder:       mockFinder,
			Logger:       logr.Discard(),
			OwnersHelper: mockOwnersHelper,
			UserHelper:   mockUserHelper,
			IssueHelper:  mockIssueHelper,
		}

		issue := &github.Issue{
			Number:  &issueNumber,
			HTMLURL: &issueURL,
			Body:    &body,
		}

		repo := test.NewRepo(t)
		sha, _ := test.AddEmptyCommit(t, repo, "empty")
		user := &github.User{
			Login: &o.Approvers[0],
		}

		gomock.InOrder(
			mockFinder.EXPECT().FindSHAs(body).Return([]plumbing.Hash{sha}, nil),
			mockUserHelper.EXPECT().GetCommitAuthor(ctx, sha.String()).Return(user, nil),
			mockOwnersHelper.EXPECT().IsApprover(o, *user.Login).Return(true),
			mockIssueHelper.EXPECT().Assign(ctx, issue, *user.Login).Return(nil),
		)

		err := a.handleIssue(ctx, issue, o)
		assert.NoError(t, err)
	})

	t.Run("working as expected, use is NOT an approver", func(t *testing.T) {

		var (
			ctx         = context.Background()
			issueNumber = 123
			issueURL    = "some url"
			body        = "some body"
			nonApprover = "notanapprover"
		)

		ctrl := gomock.NewController(t)

		mockFinder := markup.NewMockFinder(ctrl)
		mockUserHelper := gh.NewMockUserHelper(ctrl)
		mockOwnersHelper := owners.NewMockOwnersHelper(ctrl)
		mockIssueHelper := gh.NewMockIssueHelper(ctrl)

		a := Assign{
			Finder:       mockFinder,
			Logger:       logr.Discard(),
			OwnersHelper: mockOwnersHelper,
			UserHelper:   mockUserHelper,
			IssueHelper:  mockIssueHelper,
		}

		issue := &github.Issue{
			Number:  &issueNumber,
			HTMLURL: &issueURL,
			Body:    &body,
		}

		repo := test.NewRepo(t)
		sha, _ := test.AddEmptyCommit(t, repo, "empty")
		user := &github.User{
			Login: &nonApprover,
		}

		gomock.InOrder(
			mockFinder.EXPECT().FindSHAs(body).Return([]plumbing.Hash{sha}, nil),
			mockUserHelper.EXPECT().GetCommitAuthor(ctx, sha.String()).Return(user, nil),
			mockOwnersHelper.EXPECT().IsApprover(o, *user.Login).Return(false),
			mockOwnersHelper.EXPECT().GetRandomApprover(o).Return(*user.Login, nil),
			mockIssueHelper.EXPECT().Assign(ctx, issue, *user.Login).Return(nil),
		)

		err := a.handleIssue(ctx, issue, o)
		assert.NoError(t, err)
	})
}

func TestAssign_assignIssues(t *testing.T) {

	const (
		repoOwner          = "owner"
		repoName           = "repo"
		upstreamMainBranch = "us-main"
		upstreamURL        = "some-upstream-url"
		ownersFileName     = "OWNERS"
	)

	var o = &owners.Owners{
		Approvers: []string{
			"approver1",
		},
	}

	t.Run("failed to get owners from file", func(t *testing.T) {

		var (
			ctx = context.Background()
		)

		ctrl := gomock.NewController(t)

		mockFinder := markup.NewMockFinder(ctrl)
		mockGitHelper := gitutils.NewMockHelper(ctrl)
		mockIssueHelper := gh.NewMockIssueHelper(ctrl)
		mockUserHelper := gh.NewMockUserHelper(ctrl)
		mockOwnersHelper := owners.NewMockOwnersHelper(ctrl)

		ghRepoName := &gh.RepoName{
			Owner: repoOwner,
			Repo:  repoName,
		}

		upstreamConfig := config.Upstream{
			Ref: upstreamMainBranch,
			URL: upstreamURL,
		}

		downstreamConfig := config.Downstream{
			OwnersFile: ownersFileName,
		}

		a := Assign{
			Finder:           mockFinder,
			GitHelper:        mockGitHelper,
			Logger:           logr.Discard(),
			IssueHelper:      mockIssueHelper,
			UserHelper:       mockUserHelper,
			Repo:             test.NewRepo(t),
			RepoName:         ghRepoName,
			UpstreamConfig:   upstreamConfig,
			DownstreamConfig: downstreamConfig,
			OwnersHelper:     mockOwnersHelper,
		}

		gomock.InOrder(
			mockOwnersHelper.EXPECT().FromFile(a.DownstreamConfig.OwnersFile).Return(nil, errors.New("some error")),
		)

		err := a.assignIssues(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "could not get owners from file")
	})

	t.Run("failed to list open issues", func(t *testing.T) {

		var (
			ctx = context.Background()
		)

		ctrl := gomock.NewController(t)

		mockFinder := markup.NewMockFinder(ctrl)
		mockGitHelper := gitutils.NewMockHelper(ctrl)
		mockIssueHelper := gh.NewMockIssueHelper(ctrl)
		mockUserHelper := gh.NewMockUserHelper(ctrl)
		mockOwnersHelper := owners.NewMockOwnersHelper(ctrl)

		ghRepoName := &gh.RepoName{
			Owner: repoOwner,
			Repo:  repoName,
		}

		upstreamConfig := config.Upstream{
			Ref: upstreamMainBranch,
			URL: upstreamURL,
		}

		downstreamConfig := config.Downstream{
			OwnersFile: ownersFileName,
		}

		a := Assign{
			Finder:           mockFinder,
			GitHelper:        mockGitHelper,
			Logger:           logr.Discard(),
			IssueHelper:      mockIssueHelper,
			UserHelper:       mockUserHelper,
			Repo:             test.NewRepo(t),
			RepoName:         ghRepoName,
			UpstreamConfig:   upstreamConfig,
			DownstreamConfig: downstreamConfig,
			OwnersHelper:     mockOwnersHelper,
		}

		gomock.InOrder(
			mockOwnersHelper.EXPECT().FromFile(a.DownstreamConfig.OwnersFile).Return(o, nil),
			mockIssueHelper.EXPECT().ListAllOpen(ctx, true).Return([]*github.Issue{}, errors.New("some error")),
		)

		err := a.assignIssues(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "could not list open issues")
	})

	t.Run("some issues failed, continue to process other issues", func(t *testing.T) {

		var (
			ctx          = context.Background()
			issue1Number = 123
			issue2Number = 234
			issueURL     = "some url"
			body         = "some body"
		)

		ctrl := gomock.NewController(t)

		mockFinder := markup.NewMockFinder(ctrl)
		mockGitHelper := gitutils.NewMockHelper(ctrl)
		mockIssueHelper := gh.NewMockIssueHelper(ctrl)
		mockUserHelper := gh.NewMockUserHelper(ctrl)
		mockOwnersHelper := owners.NewMockOwnersHelper(ctrl)

		repo := test.NewRepo(t)

		ghRepoName := &gh.RepoName{
			Owner: repoOwner,
			Repo:  repoName,
		}

		upstreamConfig := config.Upstream{
			Ref: upstreamMainBranch,
			URL: upstreamURL,
		}

		downstreamConfig := config.Downstream{
			OwnersFile: ownersFileName,
		}

		a := Assign{
			Finder:           mockFinder,
			GitHelper:        mockGitHelper,
			Logger:           logr.Discard(),
			IssueHelper:      mockIssueHelper,
			UserHelper:       mockUserHelper,
			Repo:             repo,
			RepoName:         ghRepoName,
			UpstreamConfig:   upstreamConfig,
			DownstreamConfig: downstreamConfig,
			OwnersHelper:     mockOwnersHelper,
		}

		issues := []*github.Issue{
			{
				Number:  &issue1Number,
				HTMLURL: &issueURL,
				Body:    &body,
			},
			{
				Number:  &issue2Number,
				HTMLURL: &issueURL,
				Body:    &body,
			},
		}
		user := &github.User{
			Login: &o.Approvers[0],
		}

		sha, _ := test.AddEmptyCommit(t, repo, "empty")

		gomock.InOrder(
			mockOwnersHelper.EXPECT().FromFile(a.DownstreamConfig.OwnersFile).Return(o, nil),
			mockIssueHelper.EXPECT().ListAllOpen(ctx, true).Return(issues, nil),

			// issue #1
			mockFinder.EXPECT().FindSHAs(body).Return([]plumbing.Hash{sha}, nil),
			mockUserHelper.EXPECT().GetCommitAuthor(ctx, sha.String()).Return(user, nil),
			mockOwnersHelper.EXPECT().IsApprover(o, *user.Login).Return(true),
			mockIssueHelper.EXPECT().Assign(ctx, issues[0], o.Approvers[0]).Return(errors.New("some error")),

			// issue #2
			mockFinder.EXPECT().FindSHAs(body).Return([]plumbing.Hash{sha}, nil),
			mockUserHelper.EXPECT().GetCommitAuthor(ctx, sha.String()).Return(user, nil),
			mockOwnersHelper.EXPECT().IsApprover(o, *user.Login).Return(true),
			mockIssueHelper.EXPECT().Assign(ctx, issues[1], o.Approvers[0]).Return(nil),
		)

		err := a.assignIssues(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "could not assign issue 123")
	})

	t.Run("all issues failed, continue to process other issues", func(t *testing.T) {

		var (
			ctx          = context.Background()
			issue1Number = 123
			issue2Number = 234
			issueURL     = "some url"
			body         = "some body"
		)

		ctrl := gomock.NewController(t)

		mockFinder := markup.NewMockFinder(ctrl)
		mockGitHelper := gitutils.NewMockHelper(ctrl)
		mockIssueHelper := gh.NewMockIssueHelper(ctrl)
		mockUserHelper := gh.NewMockUserHelper(ctrl)
		mockOwnersHelper := owners.NewMockOwnersHelper(ctrl)

		repo := test.NewRepo(t)

		ghRepoName := &gh.RepoName{
			Owner: repoOwner,
			Repo:  repoName,
		}

		upstreamConfig := config.Upstream{
			Ref: upstreamMainBranch,
			URL: upstreamURL,
		}

		downstreamConfig := config.Downstream{
			OwnersFile: ownersFileName,
		}

		a := Assign{
			Finder:           mockFinder,
			GitHelper:        mockGitHelper,
			Logger:           logr.Discard(),
			IssueHelper:      mockIssueHelper,
			UserHelper:       mockUserHelper,
			Repo:             repo,
			RepoName:         ghRepoName,
			UpstreamConfig:   upstreamConfig,
			DownstreamConfig: downstreamConfig,
			OwnersHelper:     mockOwnersHelper,
		}

		issues := []*github.Issue{
			{
				Number:  &issue1Number,
				HTMLURL: &issueURL,
				Body:    &body,
			},
			{
				Number:  &issue2Number,
				HTMLURL: &issueURL,
				Body:    &body,
			},
		}
		user := &github.User{
			Login: &o.Approvers[0],
		}

		sha, _ := test.AddEmptyCommit(t, repo, "empty")

		gomock.InOrder(
			mockOwnersHelper.EXPECT().FromFile(a.DownstreamConfig.OwnersFile).Return(o, nil),
			mockIssueHelper.EXPECT().ListAllOpen(ctx, true).Return(issues, nil),

			// issue #1
			mockFinder.EXPECT().FindSHAs(body).Return([]plumbing.Hash{sha}, nil),
			mockUserHelper.EXPECT().GetCommitAuthor(ctx, sha.String()).Return(user, nil),
			mockOwnersHelper.EXPECT().IsApprover(o, *user.Login).Return(true),
			mockIssueHelper.EXPECT().Assign(ctx, issues[0], o.Approvers[0]).Return(errors.New("some error")),

			// issue #2
			mockFinder.EXPECT().FindSHAs(body).Return([]plumbing.Hash{sha}, nil),
			mockUserHelper.EXPECT().GetCommitAuthor(ctx, sha.String()).Return(user, nil),
			mockOwnersHelper.EXPECT().IsApprover(o, *user.Login).Return(true),
			mockIssueHelper.EXPECT().Assign(ctx, issues[1], o.Approvers[0]).Return(errors.New("some error")),
		)

		err := a.assignIssues(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "could not assign issue 123")
		assert.Contains(t, err.Error(), "could not assign issue 234")
	})

	t.Run("all issues succeeded", func(t *testing.T) {

		var (
			ctx          = context.Background()
			issue1Number = 123
			issue2Number = 234
			issueURL     = "some url"
			body         = "some body"
		)

		ctrl := gomock.NewController(t)

		mockFinder := markup.NewMockFinder(ctrl)
		mockGitHelper := gitutils.NewMockHelper(ctrl)
		mockIssueHelper := gh.NewMockIssueHelper(ctrl)
		mockUserHelper := gh.NewMockUserHelper(ctrl)
		mockOwnersHelper := owners.NewMockOwnersHelper(ctrl)

		repo := test.NewRepo(t)

		ghRepoName := &gh.RepoName{
			Owner: repoOwner,
			Repo:  repoName,
		}

		upstreamConfig := config.Upstream{
			Ref: upstreamMainBranch,
			URL: upstreamURL,
		}

		downstreamConfig := config.Downstream{
			OwnersFile: ownersFileName,
		}

		a := Assign{
			Finder:           mockFinder,
			GitHelper:        mockGitHelper,
			Logger:           logr.Discard(),
			IssueHelper:      mockIssueHelper,
			UserHelper:       mockUserHelper,
			Repo:             repo,
			RepoName:         ghRepoName,
			UpstreamConfig:   upstreamConfig,
			DownstreamConfig: downstreamConfig,
			OwnersHelper:     mockOwnersHelper,
		}

		issues := []*github.Issue{
			{
				Number:  &issue1Number,
				HTMLURL: &issueURL,
				Body:    &body,
			},
			{
				Number:  &issue2Number,
				HTMLURL: &issueURL,
				Body:    &body,
			},
		}
		user := &github.User{
			Login: &o.Approvers[0],
		}

		sha, _ := test.AddEmptyCommit(t, repo, "empty")

		gomock.InOrder(
			mockOwnersHelper.EXPECT().FromFile(a.DownstreamConfig.OwnersFile).Return(o, nil),
			mockIssueHelper.EXPECT().ListAllOpen(ctx, true).Return(issues, nil),

			// issue #1
			mockFinder.EXPECT().FindSHAs(body).Return([]plumbing.Hash{sha}, nil),
			mockUserHelper.EXPECT().GetCommitAuthor(ctx, sha.String()).Return(user, nil),
			mockOwnersHelper.EXPECT().IsApprover(o, *user.Login).Return(true),
			mockIssueHelper.EXPECT().Assign(ctx, issues[0], o.Approvers[0]).Return(nil),

			// issue #2
			mockFinder.EXPECT().FindSHAs(body).Return([]plumbing.Hash{sha}, nil),
			mockUserHelper.EXPECT().GetCommitAuthor(ctx, sha.String()).Return(user, nil),
			mockOwnersHelper.EXPECT().IsApprover(o, *user.Login).Return(true),
			mockIssueHelper.EXPECT().Assign(ctx, issues[1], o.Approvers[0]).Return(nil),
		)

		err := a.assignIssues(ctx)
		assert.NoError(t, err)
	})
}
