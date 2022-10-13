package gitutils

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-logr/logr"
)

//go:generate mockgen -source=helper.go -package=gitutils -destination=mock_helper.go

type Helper interface {
	GetRemoteRef(ctx context.Context, remoteName, branchName string) (*plumbing.Reference, error)
	PushContextWithAuth(ctx context.Context, token string) error
	RecreateRemote(ctx context.Context, remoteNAme, remoteURL string) (*git.Remote, error)
}

type HelperImpl struct {
	logger logr.Logger
	repo   *git.Repository
}

func NewHelper(repo *git.Repository, logger logr.Logger) Helper {
	return &HelperImpl{repo: repo, logger: logger}
}

func (h *HelperImpl) GetRemoteRef(ctx context.Context, remoteName, branchName string) (*plumbing.Reference, error) {
	remote, err := h.repo.Remote(remoteName)
	if err != nil {
		return nil, err
	}

	fo := git.FetchOptions{RemoteName: remoteName}

	if err := remote.FetchContext(ctx, &fo); err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return nil, fmt.Errorf("could not fetch remote %s: %v", remoteName, err)
	}

	from := plumbing.NewRemoteReferenceName(remoteName, branchName)

	ref, err := h.repo.Reference(from, true)
	if err != nil {
		return nil, fmt.Errorf("could not get the reference for %s/%s: %v", remoteName, branchName, err)
	}

	return ref, nil
}

func (h *HelperImpl) PushContextWithAuth(ctx context.Context, token string) error {
	po := git.PushOptions{
		Auth:  &http.BasicAuth{Username: token},
		Force: true,
	}

	return h.repo.PushContext(ctx, &po)
}

func (h *HelperImpl) RecreateRemote(ctx context.Context, remoteName, remoteURL string) (*git.Remote, error) {
	cfg, err := h.repo.Config()
	if err != nil {
		return nil, fmt.Errorf("could not get the repo config: %v", err)
	}

	logger := h.logger.WithValues("remote name", remoteName)

	if cfg.Remotes[remoteName] != nil {
		logger.Info("Remote already exists; deleting")
		if err := h.repo.DeleteRemote(remoteName); err != nil {
			return nil, fmt.Errorf("could not delete remote %s: %v", remoteName, err)
		}
	}

	logger.Info("Creating remote")

	rc := config.RemoteConfig{
		Name: remoteName,
		URLs: []string{remoteURL},
	}

	return h.repo.CreateRemote(&rc)
}
