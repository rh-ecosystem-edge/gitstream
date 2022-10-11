package gitutils

import (
	"context"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	"github.com/qbarrand/gitstream/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCherryPickerImpl_Run(t *testing.T) {
	const (
		markup   = "Some-Markup"
		repoPath = "/path/to/repo"
		sha      = "e3229f3c533ed51070beff092e5c7694a8ee81f0"
	)

	commands := [][]string{
		{"first", "command"},
		{"second", "command"},
	}

	ctrl := gomock.NewController(t)

	executor := NewMockExecutor(ctrl)

	logger := logr.Discard()

	cp := NewCherryPicker(markup, logger, commands...)
	cp.executor = executor

	ctx := context.Background()

	repo, fs := test.CloneCurrentRepoWithFS(t)

	commit := &object.Commit{
		Hash:    plumbing.NewHash(sha),
		Message: "Some message",
	}

	gomock.InOrder(
		executor.
			EXPECT().
			RunCommand(ctx, logger, "git", repoPath, "cherry-pick", "-n", sha).
			Do(func(_ context.Context, _ logr.Logger, _, _ string, _ ...string) {
				wt, err := repo.Worktree()
				require.NoError(t, err)

				const testFileName = "test-file"

				fd, err := fs.Create(testFileName)
				require.NoError(t, err)
				defer fd.Close()

				_, err = fd.Write([]byte("test contents"))
				require.NoError(t, err)

				_, err = wt.Add(testFileName)
				require.NoError(t, err)
			}),
		executor.EXPECT().RunCommand(ctx, logger, "first", repoPath, "command"),
		executor.EXPECT().RunCommand(ctx, logger, "second", repoPath, "command"),
	)

	err := cp.Run(ctx, repo, repoPath, commit)
	assert.NoError(t, err)

	head, err := repo.Head()
	require.NoError(t, err)

	headCommit, err := repo.CommitObject(head.Hash())
	require.NoError(t, err)
	assert.True(
		t,
		strings.HasSuffix(headCommit.Message, "Some-Markup: "+sha),
	)
}
