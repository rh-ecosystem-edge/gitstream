package gitutils

import (
	"context"
	"testing"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	"github.com/rh-ecosystem-edge/gitstream/internal/config"
	gh "github.com/rh-ecosystem-edge/gitstream/internal/github"
	"github.com/rh-ecosystem-edge/gitstream/internal/intents"
	"github.com/rh-ecosystem-edge/gitstream/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDifferImpl_GetMissingCommits(t *testing.T) {
	repo := test.NewRepo(t)

	ctrl := gomock.NewController(t)

	helper := NewMockHelper(ctrl)
	ig := intents.NewMockGetter(ctrl)

	di := NewDiffer(helper, ig, logr.Discard())

	since := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)

	repoName := gh.RepoName{
		Owner: "owner",
		Repo:  "repo",
	}

	const (
		branchName   = "main"
		dsMainBranch = "ds-main"
		remoteName   = "gs-upstream"
		remoteURL    = "remote-url"
	)

	usCfg := config.Upstream{
		Ref: branchName,
		URL: remoteURL,
	}

	ctx := context.Background()

	// upstream repo has 4 commits
	hash0, _ := test.AddEmptyCommit(t, repo, "commit 0")
	hash1, _ := test.AddEmptyCommit(t, repo, "commit 1")
	hash2, _ := test.AddEmptyCommit(t, repo, "commit 2")

	_, missingCommit := test.AddEmptyCommit(t, repo, "commit 3")

	// downstream has 3, brnch main points to hash2
	dsMainRef := plumbing.NewHashReference(plumbing.NewBranchReferenceName(dsMainBranch), hash2)

	// commit 3 is missing from downstream

	head, err := repo.Head()
	require.NoError(t, err)

	gomock.InOrder(
		helper.EXPECT().GetBranchRef(ctx, dsMainBranch).Return(dsMainRef, nil),
		ig.
			EXPECT().
			FromLocalGitRepo(ctx, repo, hash2, &since).
			Return(intents.CommitIntents{hash0: "commit from log"}, nil),
		ig.
			EXPECT().
			FromGitHubIssues(ctx, &repoName).
			Return(
				intents.CommitIntents{
					hash1: "commit from issue",
					hash2: "commit from PR",
				},
				nil),
		helper.EXPECT().RecreateRemote(ctx, remoteName, remoteURL),
		helper.EXPECT().GetRemoteRef(ctx, remoteName, branchName).Return(head, nil),
	)

	commits, err := di.GetMissingCommits(context.Background(), repo, &repoName, &since, dsMainBranch, usCfg)
	assert.NoError(t, err)

	assert.Len(t, commits, 1)
	assert.Contains(t, commits, missingCommit)
}
