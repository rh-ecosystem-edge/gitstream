package gitstream

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	"github.com/google/go-github/v47/github"
	"github.com/qbarrand/gitstream/internal/config"
	gh "github.com/qbarrand/gitstream/internal/github"
	"github.com/qbarrand/gitstream/internal/gitutils"
	"github.com/qbarrand/gitstream/internal/test"
	"github.com/stretchr/testify/assert"
)

func TestSync_Run(t *testing.T) {
	ctrl := gomock.NewController(t)

	const (
		downstreamMainBranch = "main"
		dryRun               = false
		githubToken          = "github-token"
		repoOwner            = "owner"
		repoPath             = "/repo/path"
		repoName             = "repo"
		upstreamMainBranch   = "us-main"
		upstreamURL          = "some-upstream-url"
	)

	mockCP := gitutils.NewMockCherryPicker(ctrl)
	mockCreator := gh.NewMockCreator(ctrl)
	mockDiffer := gitutils.NewMockDiffer(ctrl)
	mockHelper := gitutils.NewMockHelper(ctrl)

	ctx := context.Background()

	repo, _ := test.CloneCurrentRepoWithFS(t)
	ghRepoName := gh.RepoName{
		Owner: repoOwner,
		Repo:  repoName,
	}

	since := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	logger := logr.Discard()

	upstreamConfig := config.Upstream{
		Ref: upstreamMainBranch,
		URL: upstreamURL,
	}

	s := Sync{
		CherryPicker: mockCP,
		Creator:      mockCreator,
		Differ:       mockDiffer,
		DiffConfig: config.Diff{
			CommitsSince: &since,
		},
		DryRun:      dryRun,
		GitHelper:   mockHelper,
		GitHubToken: githubToken,
		Repo:        repo,
		RepoName:    &ghRepoName,
		DownstreamConfig: config.Downstream{
			LocalRepoPath: repoPath,
			MainBranch:    downstreamMainBranch,
		},
		Logger:         logger,
		UpstreamConfig: upstreamConfig,
	}

	const (
		sha1    = "e3229f3c533ed51070beff092e5c7694a8ee81f0"
		sha2    = "9c08d42326af62aa0f8cea021c4d37971606148f"
		branch1 = "gs-" + sha1
		branch2 = "gs-" + sha2
	)

	commit1 := &object.Commit{
		Hash: plumbing.NewHash(sha1),
		Committer: object.Signature{
			When: time.Date(2022, 5, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	commit2 := &object.Commit{
		Hash: plumbing.NewHash(sha2),
		Committer: object.Signature{
			When: time.Date(2022, 4, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	randomError := errors.New("random error")

	gomock.InOrder(
		mockDiffer.
			EXPECT().
			GetMissingCommits(ctx, repo, &ghRepoName, &since, upstreamConfig).
			Return([]*object.Commit{commit1, commit2}, nil),
		mockCP.EXPECT().Run(ctx, repo, repoPath, commit2),
		mockHelper.EXPECT().PushContextWithAuth(ctx, githubToken),
		mockCreator.
			EXPECT().
			CreatePR(ctx, branch2, downstreamMainBranch, upstreamURL, commit2).
			Return(&github.PullRequest{HTMLURL: github.String("some-string")}, nil),
		mockCP.
			EXPECT().
			Run(ctx, repo, repoPath, commit1).
			Return(randomError),
		mockCreator.
			EXPECT().
			CreateIssue(ctx, &ErrMatcher{Err: randomError}, upstreamURL, commit1).
			Return(&github.Issue{HTMLURL: github.String("some-string")}, nil),
	)

	assert.NoError(
		t,
		s.Run(ctx),
	)
}

type ErrMatcher struct {
	Err error
}

func (e *ErrMatcher) Matches(x interface{}) bool {
	err, ok := x.(error)
	if !ok {
		return false
	}

	return errors.Is(err, e.Err)
}

func (e *ErrMatcher) String() string {
	return fmt.Sprintf("any error wrapping %v", e.Err)
}
