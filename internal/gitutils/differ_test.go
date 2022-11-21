package gitutils

import (
	"context"
	"testing"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	"github.com/qbarrand/gitstream/internal/config"
	gh "github.com/qbarrand/gitstream/internal/github"
	"github.com/qbarrand/gitstream/internal/intents"
	"github.com/qbarrand/gitstream/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDifferImpl_GetMissingCommits(t *testing.T) {
	repo := test.CloneCurrentRepo(t)

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

	head, err := repo.Head()
	require.NoError(t, err)

	// First 3 commits of this repo
	logHash := plumbing.NewHash("e3229f3c533ed51070beff092e5c7694a8ee81f0")
	issueHash := plumbing.NewHash("9c08d42326af62aa0f8cea021c4d37971606148f")
	prHash := plumbing.NewHash("4159d2e86cc72a321de2ac0585952ddd5aa95039")

	// random hash
	dsMainHash := plumbing.NewHash("2cbad39aaf3854c020d2b969df3f795f5ba109ba")
	dsMainRef := plumbing.NewHashReference(plumbing.NewBranchReferenceName(dsMainBranch), dsMainHash)

	gomock.InOrder(
		helper.EXPECT().GetBranchRef(ctx, dsMainBranch).Return(dsMainRef, nil),
		ig.
			EXPECT().
			FromLocalGitRepo(ctx, repo, dsMainHash, &since).
			Return(intents.CommitIntents{logHash: "commit from log"}, nil),
		ig.
			EXPECT().
			FromGitHubIssues(ctx, &repoName).
			Return(
				intents.CommitIntents{
					issueHash: "commit from issue",
					prHash:    "commit from PR",
				},
				nil),
		helper.EXPECT().RecreateRemote(ctx, remoteName, remoteURL),
		helper.EXPECT().GetRemoteRef(ctx, remoteName, branchName).Return(head, nil),
	)

	commits, err := di.GetMissingCommits(context.Background(), repo, &repoName, &since, dsMainBranch, usCfg)
	assert.NoError(t, err)

	assert.NotEmpty(t, commits)

	// make a set for faster lookups
	set := make(map[plumbing.Hash]struct{})

	for _, c := range commits {
		set[c.Hash] = struct{}{}
	}

	assert.Contains(t, set, plumbing.NewHash("d36f7b79606934161431b255dd22158f5b903579")) // 4th commit of this repo
	assert.NotContains(t, set, logHash)
	assert.NotContains(t, set, issueHash)
	assert.NotContains(t, set, prHash)
}
