package graph

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func TestReachableFrom(t *testing.T) {
	tests := []struct {
		name              string
		commitOpts        []models.NewCommitOpts
		start             int
		expectedReachable []string
	}{
		{
			name: "follows all parents but not children",
			commitOpts: []models.NewCommitOpts{
				{Hash: "x", Parents: []string{"m"}},
				{Hash: "m", Parents: []string{"a", "b"}},
				{Hash: "b", Parents: []string{"a"}},
				{Hash: "o", Parents: []string{"a"}},
				{Hash: "a", Parents: []string{}},
			},
			start:             1,
			expectedReachable: []string{"m", "b", "a"},
		},
		{
			name: "finds parents that come before their child",
			commitOpts: []models.NewCommitOpts{
				{Hash: "p", Parents: []string{"q"}},
				{Hash: "c", Parents: []string{"p"}},
				{Hash: "q", Parents: []string{"unloaded"}},
			},
			start:             1,
			expectedReachable: []string{"p", "c", "q"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			hashPool := &utils.StringPool{}
			commits := lo.Map(test.commitOpts,
				func(opts models.NewCommitOpts, _ int) *models.Commit { return models.NewCommit(hashPool, opts) })

			reachable := ReachableFrom(commits, test.start)

			reachableHashes := lo.FilterMap(commits, func(commit *models.Commit, _ int) (string, bool) {
				return commit.Hash(), reachable.Includes(commit.HashPtr())
			})
			assert.Equal(t, test.expectedReachable, reachableHashes)
		})
	}
}
