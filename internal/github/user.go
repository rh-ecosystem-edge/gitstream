package github

import (
	"context"
	"fmt"

	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/go-github/v47/github"
)

//go:generate mockgen -source=user.go -package=github -destination=mock_user.go

type UserHelper interface {
	GetUser(ctx context.Context, commit *object.Commit) (*github.User, error)
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

func (uh *UserHelperImpl) GetUser(ctx context.Context, commit *object.Commit) (*github.User, error) {

	email := commit.Author.Email
	userSearchRes, _, err := uh.gc.Search.Users(ctx, email, nil)
	if err != nil {
		return nil, fmt.Errorf("could not get user from email %s: %v", email, err)
	}
	if len(userSearchRes.Users) != 1 {
		return nil, fmt.Errorf("there is more than 1 user associated with %s: %v", email, err)
	}
	return userSearchRes.Users[0], nil
}
