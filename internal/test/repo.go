package test

import (
	"testing"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/stretchr/testify/require"
)

func AddEmptyCommit(t *testing.T, repo *git.Repository, msg string) (plumbing.Hash, *object.Commit) {
	t.Helper()

	wt, err := repo.Worktree()
	require.NoError(t, err)

	co := git.CommitOptions{
		AllowEmptyCommits: true,
		Author: &object.Signature{
			Name:  "Unit tests",
			Email: "unit.tests@example.com",
			When:  time.Now(),
		},
	}

	sha, err := wt.Commit(msg, &co)
	require.NoError(t, err)

	commit, err := repo.CommitObject(sha)
	require.NoError(t, err)

	return sha, commit
}

func NewRepo(t *testing.T) *git.Repository {
	t.Helper()

	wt := memfs.New()

	repo, err := git.Init(memory.NewStorage(), wt)
	require.NoError(t, err)

	return repo
}

func NewRepoWithFS(t *testing.T) (*git.Repository, billy.Filesystem) {
	t.Helper()

	wt := memfs.New()

	repo, err := git.Init(memory.NewStorage(), wt)
	require.NoError(t, err)

	return repo, wt
}
