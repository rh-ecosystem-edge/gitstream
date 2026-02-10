package github

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExecuteAssignmentCommentTemplate(t *testing.T) {

	t.Run("random assignment", func(t *testing.T) {
		data := &AssignmentCommentData{
			AppName:              "gitstream",
			CommitSHAs:           []string{"abc123", "def456"},
			CommitAuthors:        []string{"user1", "user2"},
			ApproverCommitAuthors: []string{},
			AssignedUsers:        []string{"approver1"},
			AssignmentReason:     "none of the commit authors are approvers in the OWNERS file.",
			IsRandomAssignment:   true,
		}

		var buf bytes.Buffer
		err := ExecuteAssignmentCommentTemplate(&buf, data)

		assert.NoError(t, err)
		result := buf.String()
		assert.Contains(t, result, "🤖 **gitstream Assignment Explanation**")
		assert.Contains(t, result, "I randomly assigned this issue to **@approver1**")
		assert.Contains(t, result, "**Reason**: none of the commit authors are approvers in the OWNERS file.")
		assert.Contains(t, result, "**Referenced commits**: `abc123`, `def456`")
		assert.Contains(t, result, "**Commit authors found**: @user1, @user2")
	})

	t.Run("approver assignment", func(t *testing.T) {
		data := &AssignmentCommentData{
			AppName:              "gitstream",
			CommitSHAs:           []string{"abc123"},
			CommitAuthors:        []string{"user1", "approver1"},
			ApproverCommitAuthors: []string{"approver1"},
			AssignedUsers:        []string{"approver1"},
			AssignmentReason:     "they are the author of a referenced commit and an approver.",
			IsRandomAssignment:   false,
		}

		var buf bytes.Buffer
		err := ExecuteAssignmentCommentTemplate(&buf, data)

		assert.NoError(t, err)
		result := buf.String()
		assert.Contains(t, result, "🤖 **gitstream Assignment Explanation**")
		assert.Contains(t, result, "I assigned this issue to **@approver1** because they are the author of a referenced commit and an approver.")
		assert.Contains(t, result, "**Referenced commits**: `abc123`")
		assert.Contains(t, result, "**All commit authors**: @user1, @approver1")
		assert.Contains(t, result, "**Authors who are approvers**: @approver1")
		assert.NotContains(t, result, "randomly")
	})

	t.Run("multiple approvers assignment", func(t *testing.T) {
		data := &AssignmentCommentData{
			AppName:              "gitstream",
			CommitSHAs:           []string{"abc123", "def456"},
			CommitAuthors:        []string{"approver1", "approver2"},
			ApproverCommitAuthors: []string{"approver1", "approver2"},
			AssignedUsers:        []string{"approver1", "approver2"},
			AssignmentReason:     "they are authors of referenced commits and approvers.",
			IsRandomAssignment:   false,
		}

		var buf bytes.Buffer
		err := ExecuteAssignmentCommentTemplate(&buf, data)

		assert.NoError(t, err)
		result := buf.String()
		assert.Contains(t, result, "I assigned this issue to **@approver1, @approver2** because they are authors of referenced commits and approvers.")
		assert.Contains(t, result, "**Referenced commits**: `abc123`, `def456`")
		assert.Contains(t, result, "**All commit authors**: @approver1, @approver2")
		assert.Contains(t, result, "**Authors who are approvers**: @approver1, @approver2")
	})

	t.Run("no commit SHAs", func(t *testing.T) {
		data := &AssignmentCommentData{
			AppName:              "gitstream",
			CommitSHAs:           []string{},
			CommitAuthors:        []string{},
			ApproverCommitAuthors: []string{},
			AssignedUsers:        []string{"approver1"},
			AssignmentReason:     "they were randomly selected.",
			IsRandomAssignment:   true,
		}

		var buf bytes.Buffer
		err := ExecuteAssignmentCommentTemplate(&buf, data)

		assert.NoError(t, err)
		result := buf.String()
		assert.Contains(t, result, "I randomly assigned this item to **@approver1**")
		assert.NotContains(t, result, "Referenced commits")
		assert.NotContains(t, result, "Commit authors")
	})

	t.Run("template properly handles PRs and issues", func(t *testing.T) {
		// For PRs and issues with SHAs, it should say "issue"
		// For other items without SHAs, it should say "item"
		dataWithSHAs := &AssignmentCommentData{
			AppName:              "gitstream",
			CommitSHAs:           []string{"abc123"},
			CommitAuthors:        []string{"user1"},
			ApproverCommitAuthors: []string{"user1"},
			AssignedUsers:        []string{"user1"},
			AssignmentReason:     "they are the author of a referenced commit and an approver.",
			IsRandomAssignment:   false,
		}

		var buf bytes.Buffer
		err := ExecuteAssignmentCommentTemplate(&buf, dataWithSHAs)

		assert.NoError(t, err)
		result := buf.String()
		assert.Contains(t, result, "I assigned this issue to **@user1**")
	})
}