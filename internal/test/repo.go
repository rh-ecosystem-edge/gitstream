package test

import (
	"testing"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/stretchr/testify/require"
)

func CloneCurrentRepo(t *testing.T) *git.Repository {
	t.Helper()

	repo, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{URL: "../.."})
	require.NoError(t, err)

	return repo
}

func CloneCurrentRepoWithFS(t *testing.T) (*git.Repository, billy.Filesystem) {
	t.Helper()

	wt := memfs.New()

	repo, err := git.Clone(memory.NewStorage(), wt, &git.CloneOptions{URL: "../.."})
	require.NoError(t, err)

	return repo, wt
}
