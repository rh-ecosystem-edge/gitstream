package gitutils

import (
	"context"
	"fmt"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-logr/logr"
	"github.com/qbarrand/gitstream/internal/intents"
)

type HashSet map[plumbing.Hash]*object.Commit

type RepoWithRef struct {
	repo       *git.Repository
	remoteName string
	refName    string
}

type Differ interface {
	GetMissingCommits(ctx context.Context, repo *git.Repository, froms *plumbing.Reference, since *time.Time, intents intents.CommitIntents) (HashSet, error)
}

type DifferImpl struct {
	logger logr.Logger
}

func NewDiffer(logger logr.Logger) *DifferImpl {
	return &DifferImpl{logger: logger}
}

func (d *DifferImpl) GetMissingCommits(
	ctx context.Context,
	repo *git.Repository,
	from *plumbing.Reference,
	since *time.Time,
	intents intents.CommitIntents,
) (HashSet, error) {
	set := make(HashSet)

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

		origin, ok := intents[hash]
		if ok {
			d.logger.Info("Upstream commit found in downstream", "SHA", hash, "origin", origin)
		} else {
			d.logger.Info("Upstream commit not in downstream", "SHA", hash)
			set[hash] = commit
		}

		return nil
	})

	return set, err
}
