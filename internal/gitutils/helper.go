package gitutils

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-logr/logr"
)

//go:generate mockgen -source=helper.go -package=gitutils -destination=mock_helper.go

type Helper interface {
	FetchRemoteContext(ctx context.Context, remoteName string) error
	GetBranchRef(ctx context.Context, branchName string) (*plumbing.Reference, error)
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

func (h *HelperImpl) FetchRemoteContext(ctx context.Context, remoteName string) error {
	remote, err := h.repo.Remote(remoteName)
	if err != nil {
		return fmt.Errorf("could not find remote %s: %v", remoteName, err)
	}

	fo := git.FetchOptions{RemoteName: remoteName}

	if err := remote.FetchContext(ctx, &fo); err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return fmt.Errorf("could not fetch remote %s: %v", remoteName, err)
	}

	return nil
}

func (h *HelperImpl) GetBranchRef(ctx context.Context, branchName string) (*plumbing.Reference, error) {
	return h.repo.Reference(plumbing.NewBranchReferenceName(branchName), true)
}

func (h *HelperImpl) GetRemoteRef(ctx context.Context, remoteName, branchName string) (*plumbing.Reference, error) {
	if err := h.FetchRemoteContext(ctx, remoteName); err != nil {
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
		Auth:  AuthFromToken(token),
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

func AuthFromToken(token string) transport.AuthMethod {
	return &http.BasicAuth{Username: token}
}
