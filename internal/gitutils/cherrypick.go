package gitutils

import (
	"context"
	"errors"
	"fmt"
	"os/exec"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-logr/logr"
	"github.com/qbarrand/gitstream/internal/process"
)

//go:generate mockgen -source=cherrypick.go -package=gitutils -destination=mock_cherrypick.go

type CherryPicker interface {
	Run(ctx context.Context, repo *git.Repository, repoPath string, commit *object.Commit) error
}

type CherryPickerImpl struct {
	beforeCommitCmds [][]string
	executor         Executor
	logger           logr.Logger
	markup           string
}

func NewCherryPicker(markup string, logger logr.Logger, beforeCommitCmds ...[]string) *CherryPickerImpl {
	return &CherryPickerImpl{
		beforeCommitCmds: beforeCommitCmds,
		executor:         defaultExecutor,
		logger:           logger,
		markup:           markup,
	}
}

func (c *CherryPickerImpl) Run(ctx context.Context, repo *git.Repository, repoPath string, commit *object.Commit) error {
	sha := commit.Hash.String()

	logger := c.logger.WithValues("sha", sha)

	if err := c.executor.RunCommand(ctx, logger, "git", repoPath, "cherry-pick", "-n", sha); err != nil {
		return fmt.Errorf("error running git: %w", err)
	}

	for i, command := range c.beforeCommitCmds {
		if err := c.executor.RunCommand(ctx, logger, command[0], repoPath, command[1:]...); err != nil {
			return fmt.Errorf("could not run a command %d before committing: %v", i, err)
		}
	}

	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("could not get worktree: %v", err)
	}

	opts := git.CommitOptions{
		All:    true,
		Author: &commit.Author,
	}

	msg := fmt.Sprintf("%s\n\n%s: %v", commit.Message, c.markup, sha)

	newCommit, err := wt.Commit(msg, &opts)
	if err != nil {
		return fmt.Errorf("could not commit: %v", err)
	}

	logger.Info("Successfully committed", "new sha", newCommit)

	return nil
}

type Executor interface {
	RunCommand(ctx context.Context, logger logr.Logger, bin, dir string, args ...string) error
}

type ExecutorImpl struct {
	execContext func(ctx context.Context, bin string, args ...string) *exec.Cmd
}

var defaultExecutor Executor = &ExecutorImpl{execContext: exec.CommandContext}

func (e *ExecutorImpl) RunCommand(ctx context.Context, logger logr.Logger, bin, dir string, args ...string) error {
	cmd := e.execContext(ctx, bin, args...)
	cmd.Dir = dir

	logger.Info("Running command", "command", cmd)

	cmdOut, err := cmd.CombinedOutput()
	if err != nil {
		ee := &exec.ExitError{}

		if errors.As(err, &ee) {
			return process.NewError(ee, cmdOut, cmd.String())
		}

		return fmt.Errorf("error while running %q: %v", cmd, err)
	}

	logger.V(1).Info("Process exited normally", "output", string(cmdOut))

	return nil
}
