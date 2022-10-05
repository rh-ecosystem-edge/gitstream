package gitutils

import (
	"context"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/go-logr/logr"
	"github.com/qbarrand/gitstream/internal/intents"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDifferImpl_GetMissingCommits(t *testing.T) {
	repo, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{URL: "../.."})
	require.NoError(t, err)

	iter, err := repo.Log(&git.LogOptions{})
	require.NoError(t, err)

	err = iter.ForEach(func(commit *object.Commit) error {
		t.Log(commit.Hash)
		t.Log(commit.Message)

		return nil
	})

	di := NewDiffer(logr.Discard())

	refName := plumbing.NewBranchReferenceName("main")

	ref, err := repo.Reference(refName, true)
	require.NoError(t, err)

	since := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)

	hash1 := plumbing.NewHash("d36f7b79606934161431b255dd22158f5b903579")
	hash2 := plumbing.NewHash("4159d2e86cc72a321de2ac0585952ddd5aa95039")

	ci := intents.CommitIntents{
		hash1: "some-commit",
		hash2: "some-commit",
	}

	hs, err := di.GetMissingCommits(context.Background(), repo, ref, &since, ci)
	assert.NoError(t, err)

	assert.NotEmpty(t, hs)
	assert.NotContains(t, hs, hash1)
	assert.NotContains(t, hs, hash2)
}
