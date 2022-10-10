package gitutils

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/go-logr/logr"
	"github.com/qbarrand/gitstream/internal/intents"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDifferImpl_GetMissingCommits(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(filename), "..", "..")

	t.Logf("Opening the git repository in %s", projectRoot)

	repo, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{URL: projectRoot})
	require.NoError(t, err)

	di := NewDiffer(logr.Discard())

	head, err := repo.Head()
	require.NoError(t, err)

	since := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)

	hash1 := plumbing.NewHash("d36f7b79606934161431b255dd22158f5b903579")
	hash2 := plumbing.NewHash("4159d2e86cc72a321de2ac0585952ddd5aa95039")

	ci := intents.CommitIntents{
		hash1: "some-commit",
		hash2: "some-commit",
	}

	hs, err := di.GetMissingCommits(context.Background(), repo, head, &since, ci)
	assert.NoError(t, err)

	assert.NotEmpty(t, hs)
	assert.NotContains(t, hs, hash1)
	assert.NotContains(t, hs, hash2)
}
