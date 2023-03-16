package github

import (
	"context"
	"fmt"
	"net/url"
	"path"
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

func ParseURL(s string) (*RepoName, error) {

	u, err := url.Parse(s)
	if err != nil {
		return nil, fmt.Errorf("could not parse URL %q: %v", s, err)
	}
	items := strings.Split(u.Path, "/")
	if len(items) < 2 {
		return nil, fmt.Errorf("could not parse the URL ; format is http[s]://github.com/owner/repo")
	}

	// Items will probably looks like {github.com, owner, repo} so we need indexes 1 and 2.
	return &RepoName{Owner: items[1], Repo: items[2]}, nil
}

func (rn *RepoName) String() string {
	return path.Join(rn.Owner, rn.Repo)
}

func NewGitHubClient(ctx context.Context, token string) *github.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)

	tc := oauth2.NewClient(ctx, ts)

	return github.NewClient(tc)
}
