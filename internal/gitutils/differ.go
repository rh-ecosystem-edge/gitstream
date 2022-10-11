package gitutils

import (
	"context"
	"fmt"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-logr/logr"
	"github.com/qbarrand/gitstream/internal"
	"github.com/qbarrand/gitstream/internal/config"
	gh "github.com/qbarrand/gitstream/internal/github"
	"github.com/qbarrand/gitstream/internal/intents"
)

const upstreamRemoteName = internal.GitStreamPrefix + "upstream"

//go:generate mockgen -source=differ.go -package=gitutils -destination=mock_differ.go

type Differ interface {
	GetMissingCommits(ctx context.Context, repo *git.Repository, repoName *gh.RepoName, since *time.Time, upstreamConfig config.Upstream) ([]*object.Commit, error)
}

type DifferImpl struct {
	helper        Helper
	intentsGetter intents.Getter
	logger        logr.Logger
}

func NewDiffer(helper Helper, ig intents.Getter, logger logr.Logger) *DifferImpl {
	return &DifferImpl{
		helper:        helper,
		intentsGetter: ig,
		logger:        logger,
	}
}

func (d *DifferImpl) GetMissingCommits(
	ctx context.Context,
	repo *git.Repository,
	repoName *gh.RepoName,
	since *time.Time,
	upstreamConfig config.Upstream,
) ([]*object.Commit, error) {
	logIntents, err := d.intentsGetter.FromLocalGitRepo(ctx, repo, since)
	if err != nil {
		return nil, fmt.Errorf("could not get hashes from commits: %v", err)
	}

	issueIntents, err := d.intentsGetter.FromGitHubIssues(ctx, repoName)
	if err != nil {
		return nil, fmt.Errorf("could not get hashes from issues: %v", err)
	}

	prIntents, err := d.intentsGetter.FromGitHubOpenPRs(ctx, repoName)
	if err != nil {
		return nil, fmt.Errorf("could not get hashes from PRs: %v", err)
	}

	downstreamIntents := intents.MergeCommitIntents(
		logIntents,
		issueIntents,
		prIntents,
	)

	if _, err = d.helper.RecreateRemote(ctx, upstreamRemoteName, upstreamConfig.URL); err != nil {
		return nil, fmt.Errorf("could not recreate remote: %v", err)
	}

	from, err := d.helper.GetRemoteRef(ctx, upstreamRemoteName, upstreamConfig.Ref)
	if err != nil {
		return nil, fmt.Errorf("could not get the ref for %s/%s: %v", upstreamRemoteName, upstreamConfig.Ref, err)
	}

	commits := make([]*object.Commit, 0)

	lo := git.LogOptions{
		From: from.Hash(),
		//Order: git.LogOrderCommitterTime,
		Since: since,
	}

	iter, err := repo.Log(&lo)
	if err != nil {
		return nil, fmt.Errorf("could not get a commit iterator: %v", err)
	}

	err = iter.ForEach(func(commit *object.Commit) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		hash := commit.Hash

		origin, ok := downstreamIntents[hash]
		if ok {
			d.logger.Info("Upstream commit found in downstream", "SHA", hash, "origin", origin)
		} else {
			d.logger.Info("Upstream commit not in downstream", "SHA", hash)
			commits = append(commits, commit)
		}

		return nil
	})

	return commits, err
}
