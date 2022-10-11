package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v47/github"
	"golang.org/x/oauth2"
)

type RepoName struct {
	Owner string
	Repo  string
}

func ParseRepoName(s string) (*RepoName, error) {
	items := strings.Split(s, "/")

	if len(items) != 2 {
		return nil, fmt.Errorf("could not parse the repo name; format is owner/repo")
	}

	return &RepoName{Owner: items[0], Repo: items[1]}, nil
}

type Client struct {
	gc *github.Client
}

func NewGitHubClient(ctx context.Context, token string) *github.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)

	tc := oauth2.NewClient(ctx, ts)

	return github.NewClient(tc)
}
