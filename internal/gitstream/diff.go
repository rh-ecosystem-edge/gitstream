package gitstream

import (
	"context"
	"fmt"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-logr/logr"
	gh "github.com/qbarrand/gitstream/internal/github"
	"github.com/qbarrand/gitstream/internal/gitutils"
)

type Diff struct {
	Differ                  gitutils.Differ
	DownstreamIntentsGetter gitutils.DownstreamIntentsGetter
	DownstreamRepoName      *gh.RepoName
	GitHelper               gitutils.Helper
	Logger                  logr.Logger
	Repo                    *git.Repository
	UpstreamSince           *time.Time
	UpstreamRef             string
	UpstreamURL             string
}

func (d *Diff) Run(ctx context.Context) error {
	downstreamIntents, err := d.DownstreamIntentsGetter.GetIntents(ctx, d.Repo, d.UpstreamSince, d.DownstreamRepoName)
	if err != nil {
		return fmt.Errorf("could not get downstream commit intents: %v", err)
	}

	if err := prepareRemote(ctx, d.Repo, d.UpstreamURL, d.GitHelper); err != nil {
		return fmt.Errorf("could not prepare the remote pointing to %s: %v", d.UpstreamURL, err)
	}

	ref, err := getRefFromRemote(d.Repo, upstreamRemoteName, d.UpstreamRef)
	if err != nil {
		return fmt.Errorf("could not get the reference for %s/%s: %v", upstreamRemoteName, d.UpstreamRef, err)
	}

	diff, err := d.Differ.GetMissingCommits(ctx, d.Repo, ref, d.UpstreamSince, downstreamIntents)
	if err != nil {
		return fmt.Errorf("could not get commits not present in downstream: %v", err)
	}

	for h, c := range diff {
		d.Logger.Info(
			"Commit present upstream but not downstream",
			"sha", h,
			"message", c.Message)
	}

	return nil
}
