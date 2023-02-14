package gitstream

import (
	"fmt"
	"math/rand"

	"golang.org/x/exp/slices"
)

type Owners struct {
	Approvers []string `yaml:"approvers"`
	Reviewers []string `yaml:"reviewers"`
	Component string   `yaml:"component"`
}

func (o *Owners) contains(userLogin string) bool {

	return slices.Contains(o.Approvers, userLogin)
}

func (o *Owners) getRandom() (string, error) {

	numApprovers := len(o.Approvers)

	// rand.Intn will panic if the operrand is <= 0
	if numApprovers <= 0 {
		return "", fmt.Errorf("There is no approvers in the %s file", ownersFile)
	}

	rand := rand.Intn(len(o.Approvers))
	return o.Approvers[rand], nil
}
