package github

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/go-github/v47/github"
)

//go:generate mockgen -source=user.go -package=github -destination=mock_user.go

type UserHelper interface {
	GetCommitAuthor(ctx context.Context, sha string) (*github.User, error)
}

type UserHelperImpl struct {
	gc       *github.Client
	repoName *RepoName
}

func NewUserHelper(gc *github.Client, repoName *RepoName) UserHelper {

	return &UserHelperImpl{
		gc:       gc,
		repoName: repoName,
	}
}

var ErrUnexpectedReply = errors.New("the number of found commits isn't exactly 1")

func (uh *UserHelperImpl) GetCommitAuthor(ctx context.Context, sha string) (*github.User, error) {

	q := fmt.Sprintf("hash:%s+repo:%s/%s", sha, uh.repoName.Owner, uh.repoName.Repo)
	commitSearchRes, _, err := uh.gc.Search.Commits(ctx, q, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit %s using query %q: %v", sha, q, err)
	}
	if numCommits := *commitSearchRes.Total; numCommits != 1 {
		return nil, fmt.Errorf("%w: there are %v commits matching the search query %q", ErrUnexpectedReply, numCommits, q)
	}
	return commitSearchRes.Commits[0].Author, nil
}
