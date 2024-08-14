package gitstream

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-logr/logr"
	"github.com/rh-ecosystem-edge/gitstream/internal"
	"github.com/rh-ecosystem-edge/gitstream/internal/gitutils"
)

type DeleteRemoteBranches struct {
	GitHubToken string
	Logger      logr.Logger
	Repo        *git.Repository
}

func (d *DeleteRemoteBranches) Run(ctx context.Context) error {
	const remoteName = "origin"

	remote, err := d.Repo.Remote(remoteName)
	if err != nil {
		return fmt.Errorf("could not get remote %s: %v", remoteName, err)
	}

	auth := gitutils.AuthFromToken(d.GitHubToken)

	refs, err := remote.ListContext(ctx, &git.ListOptions{Auth: auth})
	if err != nil {
		return fmt.Errorf("could not list remote refs: %v", err)
	}

	refSpecs := make([]config.RefSpec, 0, len(refs))

	for _, r := range refs {
		refName := r.Name()

		if !refName.IsBranch() {
			continue
		}

		shortName := refName.Short()

		if strings.HasPrefix(shortName, internal.GitStreamPrefix) {
			d.Logger.Info(
				"Adding remote ref to the list of refspecs to delete",
				"name", refName,
				"short name", shortName,
			)

			refSpecs = append(
				refSpecs,
				config.RefSpec(":"+refName),
			)
		}
	}

	d.Logger.Info("Deleting remote branches", "count", len(refSpecs))

	po := git.PushOptions{
		RemoteName: remoteName,
		RefSpecs:   refSpecs,
		Auth:       auth,
		Progress:   os.Stdout,
		Prune:      true,
	}

	if err = remote.PushContext(ctx, &po); err != nil {
		return fmt.Errorf("could not push: %v", err)
	}

	return nil
}
