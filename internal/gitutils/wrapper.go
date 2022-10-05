package gitutils

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-logr/logr"
)

type RepoWrapper struct {
	*git.Repository

	logger logr.Logger

	Path  string
	Token string
}

func (rw *RepoWrapper) CherryPick(ctx context.Context, commit *object.Commit) ([]byte, error) {
	sha := commit.Hash.String()

	cmdShow := newCommand(ctx, rw.Path, "show", sha)

	stdOut, err := cmdShow.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("could not get a pipe to stdout: %v", err)
	}

	rw.logger.WithValues(
		"Running git show",
		"command", cmdShow.String(),
		"dir", rw.Path,
	)

	if err := cmdShow.Start(); err != nil {
		return nil, fmt.Errorf("could not start git show: %v", err)
	}

	cmdApply := newCommand(ctx, rw.Path, "apply", "--allow-empty", "-")
	cmdApply.Stdin = stdOut

	rw.logger.WithValues(
		"Running git apply",
		"command", cmdApply.String(),
		"dir", rw.Path,
	)

	out, err := cmdApply.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("error while applying the patch: %v", err)
	}

	if err := cmdShow.Wait(); err != nil {
		return nil, fmt.Errorf("error while waiting for git show: %v", err)
	}

	msg := fmt.Sprintf("%s\n\nUpstream-Commit: %v", commit.Message, sha)

	wt, err := rw.Worktree()
	if err != nil {
		return nil, fmt.Errorf("could not get the work tree: %v", err)
	}

	opts := git.CommitOptions{
		All:    true,
		Author: &commit.Author,
	}

	if _, err := wt.Commit(msg, &opts); err != nil {
		return nil, fmt.Errorf("could not commit: %v", err)
	}

	return out, nil
}

func newCommand(ctx context.Context, dir string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir

	return cmd
}
