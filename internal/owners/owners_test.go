package owners

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slices"
)

func TestOwnersHelper_FromFile(t *testing.T) {

	t.Run("failed to open file", func(t *testing.T) {

		_, err := NewOwnersHelper().FromFile("nosuchfile")

		assert.Error(t, err)
		assert.ErrorContains(t, err, "could not read file")
	})

	t.Run("failed to parse file", func(t *testing.T) {

		_, err := NewOwnersHelper().FromFile("testdata/malformed_owners.yaml")

		assert.Error(t, err)
		assert.ErrorContains(t, err, "could not decode file")
	})

	t.Run("working as expected", func(t *testing.T) {

		o, err := NewOwnersHelper().FromFile("testdata/owners.yaml")

		assert.NoError(t, err)
		assert.True(t, slices.Contains(o.Approvers, "approver1"))
	})
}

func TestOwnersHelper_IsApprover(t *testing.T) {

	const (
		approver    = "user1"
		nonApprover = "user2"
	)

	o := &Owners{
		Approvers: []string{
			approver,
		},
		Component: "Some Component",
	}

	t.Run("user is not an approver", func(t *testing.T) {

		res := NewOwnersHelper().IsApprover(o, nonApprover)
		assert.False(t, res)
	})

	t.Run("user is an approver", func(t *testing.T) {

		res := NewOwnersHelper().IsApprover(o, approver)
		assert.True(t, res)
	})
}

func TestOwnersHelper_GetRandomApprover(t *testing.T) {

	t.Run("invalid approvers", func(t *testing.T) {

		o := &Owners{
			Approvers: []string{},
			Component: "Some Component",
		}

		_, err := NewOwnersHelper().GetRandomApprover(o)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "There are no approvers in owners")
	})

	t.Run("working as expected", func(t *testing.T) {

		const approver = "user1"

		o := &Owners{
			Approvers: []string{
				approver,
			},
			Component: "Some Component",
		}

		randApprover, err := NewOwnersHelper().GetRandomApprover(o)

		assert.NoError(t, err)
		assert.Equal(t, randApprover, approver)
	})
}
