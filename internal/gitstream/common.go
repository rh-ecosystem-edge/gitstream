package gitstream

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/qbarrand/gitstream/internal"
	"github.com/qbarrand/gitstream/internal/gitutils"
)

const upstreamRemoteName = internal.GitStreamPrefix + "upstream"

func getRefFromRemote(repo *git.Repository, remoteName, refName string) (*plumbing.Reference, error) {
	from := plumbing.NewRemoteReferenceName(remoteName, refName)

	ref, err := repo.Reference(from, true)
	if err != nil {
		return nil, fmt.Errorf("could not get the reference for %s/%s: %v", remoteName, refName, err)
	}

	return ref, nil
}

func prepareRemote(ctx context.Context, repo *git.Repository, upstreamURL string, gitHelper gitutils.Helper) error {
	upstreamRemoteConfig := config.RemoteConfig{
		Name: upstreamRemoteName,
		URLs: []string{upstreamURL},
	}

	remote, err := gitHelper.RecreateRemote(repo, &upstreamRemoteConfig)
	if err != nil {
		return fmt.Errorf("could not recreate remote %s", upstreamRemoteName)
	}

	if err := remote.FetchContext(ctx, &git.FetchOptions{RemoteName: upstreamRemoteName}); err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return fmt.Errorf("could not fetch remote %s: %v", upstreamRemoteName, err)
	}

	return nil
}
