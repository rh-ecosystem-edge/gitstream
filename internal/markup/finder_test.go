package markup_test

import (
	"testing"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/rh-ecosystem-edge/gitstream/internal/markup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFinder(t *testing.T) {
	t.Run("invalid key", func(t *testing.T) {
		_, err := markup.NewFinder("[")
		assert.Error(t, err)
	})

	t.Run("valid key", func(t *testing.T) {
		_, err := markup.NewFinder("Upstream-Commit")
		assert.NoError(t, err)
	})
}

func TestFinder_FindSHAs(t *testing.T) {
	cases := []struct {
		name string
		text string
		shas []plumbing.Hash
	}{
		{
			name: "0 matches",
			text: "",
			shas: make([]plumbing.Hash, 0),
		},
		{
			name: "1 match",
			text: "Some-Key: a109a5cfd36f7abe14089da2da0638149c4dc6cc",
			shas: []plumbing.Hash{
				plumbing.NewHash("a109a5cfd36f7abe14089da2da0638149c4dc6cc"),
			},
		},
		{
			name: "1 match after a newline",
			text: `Some-unrelated text

Some-Key: a109a5cfd36f7abe14089da2da0638149c4dc6cc
`,
			shas: []plumbing.Hash{
				plumbing.NewHash("a109a5cfd36f7abe14089da2da0638149c4dc6cc"),
			},
		},
		{
			name: "2 matches",
			text: `
Some-unrelated text
Some-Key: a109a5cfd36f7abe14089da2da0638149c4dc6cc
Some-Key: a109a5cfd36f7abe14089da2da0638149c4dc6cd
Invalid line Some-Key: a109a5cfd36f7abe14089da2da0638149c4dc6cd
Some-Key: a109a5cfd36f7abe14089da2da0638149c4dc6cd another invalid line
`,
			shas: []plumbing.Hash{
				plumbing.NewHash("a109a5cfd36f7abe14089da2da0638149c4dc6cc"),
				plumbing.NewHash("a109a5cfd36f7abe14089da2da0638149c4dc6cd"),
			},
		},
	}

	f, err := markup.NewFinder("Some-Key")
	require.NoError(t, err)

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			shas, err := f.FindSHAs(c.text)
			assert.NoError(t, err)
			assert.Equal(t, c.shas, shas)
		})
	}
}
