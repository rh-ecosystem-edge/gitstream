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
		Reviewers: []string{
			"reviewer1",
			"reviewer2",
		},
		Component: "Some component",
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

	t.Run("do nothing if the issue is already assigned", func(t *testing.T) {

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
			DryRun:           false,
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
			mockOwnersHelper.EXPECT().FromFile(a.DownstreamConfig.OwnersFile).Return(o, nil),
			mockIssueHelper.EXPECT().ListAllOpen(ctx, true).Return(issues, nil),
		)

		err := a.assignIssues(ctx)
		assert.NoError(t, err)
	})

	t.Run("failed to find SHAs", func(t *testing.T) {

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
			mockOwnersHelper.EXPECT().FromFile(a.DownstreamConfig.OwnersFile).Return(o, nil),
			mockIssueHelper.EXPECT().ListAllOpen(ctx, true).Return(issues, nil),
			mockFinder.EXPECT().FindSHAs(gomock.Any()).Return([]plumbing.Hash{}, errors.New("some error")),
		)

		err := a.assignIssues(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error while looking for SHAs")
	})

	t.Run("failed to get commit author, Github API error", func(t *testing.T) {

		var (
			ctx = context.Background()
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

		sha, _ := test.AddEmptyCommit(t, repo, "empty")

		gomock.InOrder(
			mockOwnersHelper.EXPECT().FromFile(a.DownstreamConfig.OwnersFile).Return(o, nil),
			mockIssueHelper.EXPECT().ListAllOpen(ctx, true).Return(issues, nil),
			mockFinder.EXPECT().FindSHAs(body).Return([]plumbing.Hash{sha}, nil),
			mockUserHelper.EXPECT().GetCommitAuthor(ctx, sha.String()).Return(nil, errors.New("some API error")),
		)

		err := a.assignIssues(ctx)
		assert.NoError(t, err)
	})

	t.Run("failed to get commit author, commit not found", func(t *testing.T) {

		var (
			ctx = context.Background()
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

		sha, _ := test.AddEmptyCommit(t, repo, "empty")

		gomock.InOrder(
			mockOwnersHelper.EXPECT().FromFile(a.DownstreamConfig.OwnersFile).Return(o, nil),
			mockIssueHelper.EXPECT().ListAllOpen(ctx, true).Return(issues, nil),
			mockFinder.EXPECT().FindSHAs(body).Return([]plumbing.Hash{sha}, nil),
			mockUserHelper.EXPECT().GetCommitAuthor(ctx, sha.String()).Return(nil, gh.ErrUnexpectedReply),
			mockOwnersHelper.EXPECT().GetRandomApprover(o).Return(o.Approvers[0], nil),
			mockIssueHelper.EXPECT().Assign(ctx, issues[0], o.Approvers[0]).Return(nil),
		)

		err := a.assignIssues(ctx)
		assert.NoError(t, err)
	})
	//
	t.Run("user is an owner", func(t *testing.T) {

		var (
			ctx = context.Background()
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
		user := &github.User{
			Login: &o.Approvers[0],
		}

		sha, _ := test.AddEmptyCommit(t, repo, "empty")

		gomock.InOrder(
			mockOwnersHelper.EXPECT().FromFile(a.DownstreamConfig.OwnersFile).Return(o, nil),
			mockIssueHelper.EXPECT().ListAllOpen(ctx, true).Return(issues, nil),
			mockFinder.EXPECT().FindSHAs(body).Return([]plumbing.Hash{sha}, nil),
			mockUserHelper.EXPECT().GetCommitAuthor(ctx, sha.String()).Return(user, nil),
			mockOwnersHelper.EXPECT().IsApprover(o, *user.Login).Return(true),
			mockIssueHelper.EXPECT().Assign(ctx, issues[0], o.Approvers[0]).Return(nil),
		)

		err := a.assignIssues(ctx)
		assert.NoError(t, err)
	})

	t.Run("user is NOT an owner", func(t *testing.T) {

		var (
			ctx = context.Background()
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

		var (
			issueNumber = 123
			issueURL    = "some url"
			body        = "some body"
			nonApprover = "notanapprover"
		)
		issues := []*github.Issue{
			{
				Number:  &issueNumber,
				HTMLURL: &issueURL,
				Body:    &body,
			},
		}
		user := &github.User{
			Login: &nonApprover,
		}

		sha, _ := test.AddEmptyCommit(t, repo, "empty")

		gomock.InOrder(
			mockOwnersHelper.EXPECT().FromFile(a.DownstreamConfig.OwnersFile).Return(o, nil),
			mockIssueHelper.EXPECT().ListAllOpen(ctx, true).Return(issues, nil),
			mockFinder.EXPECT().FindSHAs(body).Return([]plumbing.Hash{sha}, nil),
			mockUserHelper.EXPECT().GetCommitAuthor(ctx, sha.String()).Return(user, nil),
			mockOwnersHelper.EXPECT().IsApprover(o, *user.Login).Return(false),
			mockOwnersHelper.EXPECT().GetRandomApprover(o).Return(o.Approvers[0], nil),
			mockIssueHelper.EXPECT().Assign(ctx, issues[0], o.Approvers[0]).Return(nil),
		)

		err := a.assignIssues(ctx)
		assert.NoError(t, err)
	})

	t.Run("failed to assign an issue", func(t *testing.T) {

		var (
			ctx = context.Background()
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

		user := &github.User{
			Login: &o.Approvers[0],
		}

		sha, _ := test.AddEmptyCommit(t, repo, "empty")

		gomock.InOrder(
			mockOwnersHelper.EXPECT().FromFile(a.DownstreamConfig.OwnersFile).Return(o, nil),
			mockIssueHelper.EXPECT().ListAllOpen(ctx, true).Return(issues, nil),
			mockFinder.EXPECT().FindSHAs(body).Return([]plumbing.Hash{sha}, nil),
			mockUserHelper.EXPECT().GetCommitAuthor(ctx, sha.String()).Return(user, nil),
			mockOwnersHelper.EXPECT().IsApprover(o, *user.Login).Return(true),
			mockIssueHelper.EXPECT().Assign(ctx, issues[0], *user.Login).Return(errors.New("error")),
		)

		err := a.assignIssues(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "could not assign issue")
	})

	t.Run("working as expected", func(t *testing.T) {

		var (
			ctx = context.Background()
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

		user := &github.User{
			Login: &o.Approvers[0],
		}

		sha, _ := test.AddEmptyCommit(t, repo, "empty")

		gomock.InOrder(
			mockOwnersHelper.EXPECT().FromFile(a.DownstreamConfig.OwnersFile).Return(o, nil),
			mockIssueHelper.EXPECT().ListAllOpen(ctx, true).Return(issues, nil),
			mockFinder.EXPECT().FindSHAs(body).Return([]plumbing.Hash{sha}, nil),
			mockUserHelper.EXPECT().GetCommitAuthor(ctx, sha.String()).Return(user, nil),
			mockOwnersHelper.EXPECT().IsApprover(o, *user.Login).Return(true),
			mockIssueHelper.EXPECT().Assign(ctx, issues[0], *user.Login).Return(nil),
		)

		err := a.assignIssues(ctx)
		assert.NoError(t, err)
	})
}
