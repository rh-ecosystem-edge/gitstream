package github

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseURL(t *testing.T) {

	t.Run("bad URL", func(t *testing.T) {

		_, err := ParseURL("bad url")
		assert.Error(t, err)
	})

	t.Run("working as expected", func(t *testing.T) {

		const (
			owner = "some-owner"
			repo  = "some-repo"
		)

		repoName, err := ParseURL(fmt.Sprintf("https://github.com/%s/%s", owner, repo))
		assert.NoError(t, err)
		assert.Equal(t, repoName.Owner, "some-owner")
		assert.Equal(t, repoName.Repo, "some-repo")
	})
}
