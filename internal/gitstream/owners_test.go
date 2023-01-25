package gitstream

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOwners_getAssignee(t *testing.T) {

	const (
		approver    = "user1"
		nonApprover = "user2"
	)

	owners := &Owners{
		Approvers: []string{
			approver,
		},
		Component: "Some Component",
	}

	t.Run("user not found", func(t *testing.T) {

		res := owners.contains(nonApprover)
		assert.False(t, res)
	})

	t.Run("user found", func(t *testing.T) {

		res := owners.contains(approver)
		assert.True(t, res)
	})
}

func TestOwners_getRandomAssignee(t *testing.T) {

	const (
		approver = "user1"
	)

	t.Run("empty owners file", func(t *testing.T) {

		owners := &Owners{
			Approvers: []string{},
			Component: "Some Component",
		}

		_, err := owners.getRandom()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "There is no approvers in the", "file")
	})

	t.Run("user is not an approver, a random approver is picked", func(t *testing.T) {

		owners := &Owners{
			Approvers: []string{
				approver,
			},
			Component: "Some Component",
		}

		assignee, err := owners.getRandom()

		assert.NoError(t, err)
		assert.Equal(t, assignee, approver)
	})
}
