package gitutils

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	"github.com/qbarrand/gitstream/internal/process"
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

	repo, fs := test.NewRepoWithFS(t)

	commit := &object.Commit{
		Hash:    plumbing.NewHash(sha),
		Message: "Some message",
	}

	gomock.InOrder(
		executor.
			EXPECT().
			RunCommand(ctx, logger, "git", repoPath, "cherry-pick", "-n", sha, "-m1").
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

type executorFunc = func(ctx context.Context, bin string, args ...string) *exec.Cmd

func TestExecutorImpl_RunCommand(t *testing.T) {
	const output = "some-output"

	getMockExecutor := func(ret int, dir string) executorFunc {
		return func(ctx context.Context, bin string, args ...string) *exec.Cmd {
			helperArgs := append([]string{"-test.run=TestExecutorHelper", "--", bin}, args...)

			commandLine := strings.Join(
				append([]string{bin}, args...),
				" ",
			)

			cmd := exec.CommandContext(ctx, os.Args[0], helperArgs...)
			cmd.Env = []string{
				"_TEST_HELPER_PROCESS=1",
				"_TEST_EXPECTED_DIR=" + dir,
				"_TEST_EXPECTED_COMMAND_LINE=" + commandLine,
				"_TEST_OUTPUT=some-output",
				"_TEST_RETCODE=" + strconv.Itoa(ret),
			}

			return cmd
		}
	}

	t.Run("process returns an error", func(t *testing.T) {
		tempDir := t.TempDir()

		const ret = 123

		ex := ExecutorImpl{
			execContext: getMockExecutor(123, tempDir),
		}

		err := ex.RunCommand(context.Background(), logr.Discard(), "process", tempDir, "arg1", "arg2")
		require.Error(t, err)

		pe := &process.Error{}

		assert.ErrorAs(t, err, &pe)
		assert.Equal(t, ret, pe.ExitCode())
		assert.Equal(t, []byte(output), pe.Combined())
	})

	t.Run("process returns no error", func(t *testing.T) {
		tempDir := t.TempDir()

		ex := ExecutorImpl{
			execContext: getMockExecutor(0, tempDir),
		}

		err := ex.RunCommand(context.Background(), logr.Discard(), "process", tempDir, "arg1", "arg2")
		assert.NoError(t, err)
	})
}

func TestExecutorHelper(t *testing.T) {
	if os.Getenv("_TEST_HELPER_PROCESS") != "1" {
		return
	}

	assert.Equal(
		t,
		strings.Join(os.Args[1:], " "),
		os.Getenv("_TEST_EXPECTED_COMMAND_LINE"),
	)

	wd, err := os.Getwd()
	require.NoError(t, err)
	assert.Equal(t, os.Getenv("_TEST_EXPECTED_DIR"), wd)

	retCode, err := strconv.Atoi(os.Getenv("_TEST_RETCODE"))
	require.NoError(t, err)

	fmt.Fprint(os.Stdout, os.Getenv("_TEST_OUTPUT"))
	os.Exit(retCode)
}
