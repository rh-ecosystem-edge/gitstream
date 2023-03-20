package github

import (
	"context"
	"fmt"
	"net/url"

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

func (uh *UserHelperImpl) GetCommitAuthor(ctx context.Context, sha string) (*github.User, error) {

	q := url.Values{}
	q.Add("hash", sha)
	q.Add("repo", uh.repoName.String())

	commitSearchRes, _, err := uh.gc.Search.Commits(ctx, q.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit %s using query %q: %v", sha, q, err)
	}
	if numCommits := *commitSearchRes.Total; numCommits != 1 {
		return nil, fmt.Errorf("expected 1 commit in the search results for %q, got %d", q, numCommits)
	}
	return commitSearchRes.Commits[0].Author, nil
}
