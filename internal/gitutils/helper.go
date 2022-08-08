package gitutils

import (
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-logr/logr"
)

type Helper interface {
	RecreateBranch(repo *git.Repository, bc *config.Branch) error
	RecreateRemote(repo *git.Repository, rc *config.RemoteConfig) (*git.Remote, error)
}

type HelperImpl struct {
	logger logr.Logger
}

func NewHelper(logger logr.Logger) Helper {
	return &HelperImpl{logger: logger}
}

func (h *HelperImpl) RecreateBranch(repo *git.Repository, bc *config.Branch) error {
	cfg, err := repo.Config()
	if err != nil {
		return fmt.Errorf("could not get the repo config: %v", err)
	}

	branchName := bc.Name

	logger := h.logger.WithValues("branch name", branchName)

	if cfg.Branches[branchName] != nil {
		logger.Info("Branch already exists; deleting")
		if err := repo.DeleteBranch(branchName); err != nil {
			return fmt.Errorf("could not delete branch %s: %v", branchName, err)
		}
	}

	logger.Info("Creating branch")

	return repo.CreateBranch(bc)
}

func (h *HelperImpl) RecreateRemote(repo *git.Repository, rc *config.RemoteConfig) (*git.Remote, error) {
	cfg, err := repo.Config()
	if err != nil {
		return nil, fmt.Errorf("could not get the repo config: %v", err)
	}

	remoteName := rc.Name

	logger := h.logger.WithValues("remote name", remoteName)

	if cfg.Remotes[remoteName] != nil {
		logger.Info("Remote already exists; deleting")
		if err := repo.DeleteRemote(remoteName); err != nil {
			return nil, fmt.Errorf("could not delete remote %s: %v", remoteName, err)
		}
	}

	logger.Info("Creating remote")

	return repo.CreateRemote(rc)

}
