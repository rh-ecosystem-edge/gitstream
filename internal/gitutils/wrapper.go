package gitutils

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-logr/logr"
)

type RepoWrapper struct {
	*git.Repository

	logger logr.Logger
	path   string
	token  string
}

func NewRepoWrapper(repo *git.Repository, logger logr.Logger, path, token string) *RepoWrapper {
	return &RepoWrapper{
		Repository: repo,
		logger:     logger,
		path:       path,
		token:      token,
	}
}

func (rw *RepoWrapper) CherryPick(ctx context.Context, commit *object.Commit) ([]byte, error) {
	sha := commit.Hash.String()

	cmdShow := newCommand(ctx, rw.path, "show", sha)

	stdOut, err := cmdShow.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("could not get a pipe to stdout: %v", err)
	}

	rw.logger.WithValues(
		"Running git show",
		"command", cmdShow.String(),
		"dir", rw.path,
	)

	if err := cmdShow.Start(); err != nil {
		return nil, fmt.Errorf("could not start git show: %v", err)
	}

	cmdApply := newCommand(ctx, rw.path, "apply", "--allow-empty", "-")
	cmdApply.Stdin = stdOut

	rw.logger.WithValues(
		"Running git apply",
		"command", cmdApply.String(),
		"dir", rw.path,
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

func (rw *RepoWrapper) PushContextWithAuth(ctx context.Context) error {
	po := git.PushOptions{
		Auth:  &http.BasicAuth{Username: rw.token},
		Force: true,
	}

	return rw.PushContext(ctx, &po)
}

func newCommand(ctx context.Context, dir string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir

	return cmd
}
