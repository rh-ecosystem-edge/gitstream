package markup

import (
	"fmt"
	"regexp"

	"github.com/go-git/go-git/v5/plumbing"
)

//go:generate mockgen -source=finder.go -package=markup -destination=mock_finder.go

type Finder interface {
	FindSHAs(string) ([]plumbing.Hash, error)
}

type finder struct {
	re *regexp.Regexp
}

func NewFinder(markup string) (Finder, error) {
	pattern := fmt.Sprintf(`(?m)^%s:\s*([a-z0-9]+)$`, markup)

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regexp: %v", err)
	}

	return &finder{re: re}, nil
}

func (f *finder) FindSHAs(s string) ([]plumbing.Hash, error) {
	hashes := make([]plumbing.Hash, 0)

	for _, item := range f.re.FindAllStringSubmatch(s, -1) {
		hashes = append(
			hashes,
			plumbing.NewHash(item[1]),
		)
	}

	return hashes, nil
}
