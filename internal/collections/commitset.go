package collections

import "github.com/go-git/go-git/v5/plumbing"

type CommitSet map[plumbing.Hash]bool

func NewCommitSet(elems ...plumbing.Hash) CommitSet {
	m := make(map[plumbing.Hash]bool, len(elems))

	for _, h := range elems {
		m[h] = true
	}

	return m
}

func (cs CommitSet) Add(hashes ...plumbing.Hash) {
	for _, h := range hashes {
		cs[h] = true
	}
}

func (cs CommitSet) Contains(h plumbing.Hash) bool {
	return cs[h]
}

func (cs CommitSet) Merge(ocs ...CommitSet) {
	for _, set := range ocs {
		for k := range set {
			cs[k] = true
		}
	}
}
