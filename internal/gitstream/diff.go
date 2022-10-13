package gitstream

import (
	"context"
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/go-logr/logr"
	"github.com/qbarrand/gitstream/internal/config"
	gh "github.com/qbarrand/gitstream/internal/github"
	"github.com/qbarrand/gitstream/internal/gitutils"
)

type Diff struct {
	Differ         gitutils.Differ
	DiffConfig     config.Diff
	Logger         logr.Logger
	Repo           *git.Repository
	RepoName       *gh.RepoName
	UpstreamConfig config.Upstream
}

func (d *Diff) Run(ctx context.Context) error {
	diff, err := d.Differ.GetMissingCommits(ctx, d.Repo, d.RepoName, d.DiffConfig.CommitsSince, d.UpstreamConfig)
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
