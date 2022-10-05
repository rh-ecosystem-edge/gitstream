package gitutils

import (
	"context"
	"fmt"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-logr/logr"
	"github.com/google/go-github/v47/github"
	gh "github.com/qbarrand/gitstream/internal/github"
	"github.com/qbarrand/gitstream/internal/intents"
)

type DownstreamIntentsGetter interface {
	GetIntents(ctx context.Context, repo *git.Repository, since *time.Time, repoName *gh.RepoName) (intents.CommitIntents, error)
}

type DownstreamIntentsGetterImpl struct {
	gc            *github.Client
	intentsGetter intents.Getter
	logger        logr.Logger
}

func NewDownstreamIntentsGetter(logger logr.Logger, intentsGetter intents.Getter, gc *github.Client) *DownstreamIntentsGetterImpl {
	return &DownstreamIntentsGetterImpl{
		gc:            gc,
		intentsGetter: intentsGetter,
		logger:        logger,
	}
}

func (d *DownstreamIntentsGetterImpl) GetIntents(
	ctx context.Context,
	repo *git.Repository,
	since *time.Time,
	repoName *gh.RepoName,
) (intents.CommitIntents, error) {
	logIntents, err := d.intentsGetter.FromLocalGitRepo(ctx, repo, since)
	if err != nil {
		return nil, fmt.Errorf("could not get hashes from commits: %v", err)
	}

	issueIntents, err := d.intentsGetter.FromGitHubIssues(ctx, d.gc, repoName)
	if err != nil {
		return nil, fmt.Errorf("could not get hashes from issues: %v", err)
	}

	prIntents, err := d.intentsGetter.FromGitHubOpenPRs(ctx, d.gc, repoName)
	if err != nil {
		return nil, fmt.Errorf("could not get hashes from PRs: %v", err)
	}

	allDownstreamIntents := intents.MergeCommitIntents(
		logIntents,
		issueIntents,
		prIntents,
	)

	return allDownstreamIntents, nil
}
