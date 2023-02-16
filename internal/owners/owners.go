package owners

import (
	"errors"
	"fmt"
	"math/rand"
	"os"

	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

type Owners struct {
	Approvers []string `yaml:"approvers"`
	Reviewers []string `yaml:"reviewers"`
	Component string   `yaml:"component"`
}

//go:generate mockgen -source=owners.go -package=owners -destination=mock_owners.go

type OwnersHelper interface {
	FromFile(filePath string) (*Owners, error)
	IsApprover(o *Owners, userLogin string) bool
	GetRandomApprover(o *Owners) (string, error)
}

type ownersHelper struct{}

func NewOwnersHelper() OwnersHelper {
	return &ownersHelper{}
}

func (oh *ownersHelper) FromFile(filePath string) (*Owners, error) {

	fd, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not read file %s: %v", filePath, err)
	}
	defer fd.Close()

	var o Owners
	if err := yaml.NewDecoder(fd).Decode(&o); err != nil {
		return nil, fmt.Errorf("could not decode file %s: %v", filePath, err)
	}

	return &o, nil
}

func (oh *ownersHelper) IsApprover(o *Owners, userLogin string) bool {
	return slices.Contains(o.Approvers, userLogin)
}

func (oh *ownersHelper) GetRandomApprover(o *Owners) (string, error) {

	numApprovers := len(o.Approvers)

	// rand.Intn will panic if the operrand is <= 0
	if numApprovers <= 0 {
		return "", errors.New("There are no approvers in owners")
	}

	idx := rand.Intn(numApprovers)
	return o.Approvers[idx], nil
}
