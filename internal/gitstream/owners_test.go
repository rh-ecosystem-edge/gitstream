package gitstream

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOwners_getAssignee(t *testing.T) {

	ctx := context.Background()

	owners := &Owners{
		Approvers: []string{
			"user1",
		},
		Reviewers: []string{
			"user1",
			"user2",
		},
		Component: "Kernel Module Management",
	}

	t.Run("user is an approver", func(t *testing.T) {

		var userLogin = "user1"

		assignee, err := owners.getAssignee(ctx, nil, userLogin)

		assert.NoError(t, err)
		assert.Equal(t, assignee, userLogin)
	})

	t.Run("user is not an approver, a random approver is picked", func(t *testing.T) {

		var (
			anonymousUserLogin = "jdoe"
		)

		assignee, err := owners.getAssignee(ctx, nil, anonymousUserLogin)

		assert.NoError(t, err)
		assert.NotEqual(t, anonymousUserLogin, assignee)
	})
}
