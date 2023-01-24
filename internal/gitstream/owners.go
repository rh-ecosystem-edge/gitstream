package gitstream

import (
	"context"
	"math/rand"

	"github.com/google/go-github/v47/github"
)

type Owners struct {
	Approvers []string `yaml:"approvers"`
	Reviewers []string `yaml:"reviewers"`
	Component string   `yaml:"component"`
}

func (o *Owners) getAssignee(ctx context.Context, gc *github.Client, userLogin string) (string, error) {

	for _, approver := range o.Approvers {
		if approver == userLogin {
			return userLogin, nil
		}
	}

	// User isn't in m/s OWNER file, select a random owner
	rand := rand.Intn(len(o.Approvers))
	return o.Approvers[rand], nil
}
