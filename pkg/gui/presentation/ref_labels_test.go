package presentation

import (
	"testing"

	"github.com/jesseduffield/generics/set"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
	"github.com/stefanhaller/git-todo-parser/todo"
	"github.com/stretchr/testify/assert"
)

func TestCommitRefLabels(t *testing.T) {
	tests := []struct {
		name      string
		extraInfo string
		expected  []refLabel
	}{
		{name: "no refs", extraInfo: "", expected: []refLabel{}},
		{name: "the stash is not a branch", extraInfo: "(refs/stash)", expected: []refLabel{}},
		{
			name:      "checked out branch with its upstream comes first, as one label",
			extraInfo: "(tag: v1.0, origin/feature, HEAD -> main, origin/main, feature)",
			expected: []refLabel{
				{name: "main", head: true, local: true, remote: true},
				{name: "feature", local: true, remote: true},
				{name: "v1.0", tag: true},
			},
		},
		{
			name:      "remote branch without a local one, and branch names with slashes",
			extraInfo: "(origin/hotfix/ROAR-1, hotfix/ROAR-2, origin/HEAD)",
			expected: []refLabel{
				{name: "hotfix/ROAR-2", local: true},
				{name: "origin/hotfix/ROAR-1", remote: true},
			},
		},
		{
			name:      "detached head",
			extraInfo: "(HEAD, tag: v2)",
			expected: []refLabel{
				{name: "HEAD", head: true},
				{name: "v2", tag: true},
			},
		},
	}

	localBranchNames := set.NewFromSlice([]string{"main", "feature", "hotfix/ROAR-2"})
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			commit := models.NewCommit(&utils.StringPool{}, models.NewCommitOpts{Hash: "a", ExtraInfo: test.extraInfo})

			assert.Equal(t, test.expected, commitRefLabels(commit, localBranchNames))
		})
	}
}

func TestRenderRefLabels(t *testing.T) {
	tests := []struct {
		name      string
		labels    []refLabel
		withIcons bool
		expected  string
	}{
		{name: "no labels", labels: nil, expected: "                "},
		{
			name:     "checked-out branch and a tag",
			labels:   []refLabel{{name: "main", head: true, local: true, remote: true}, {name: "v1", tag: true}},
			expected: " ✓ main +1 ─────",
		},
		{
			name:      "with icons",
			labels:    []refLabel{{name: "main", head: true, local: true, remote: true}, {name: "v1", tag: true}},
			withIcons: true,
			expected:  " ✓ main   +1 ─",
		},
		{
			name:      "tag with icon",
			labels:    []refLabel{{name: "v1.0", tag: true}},
			withIcons: true,
			expected:  " v1.0  ────────",
		},
		{
			name:     "long names are truncated so that the line to the graph still shows",
			labels:   []refLabel{{name: "feature/ROAR-13888-cleanup", local: true}},
			expected: " feature/ROAR… ─",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rendered := renderRefLabels(test.labels, 16, &style.FgDefault, &style.FgDefault, test.withIcons)

			assert.Equal(t, test.expected, utils.Decolorise(rendered))
		})
	}
}

func TestBranchNameOfCommit(t *testing.T) {
	commitOpts := []models.NewCommitOpts{
		{Hash: "branchname-todo", Parents: []string{"branchname-main"}, Action: todo.Pick},
		{Hash: "branchname-main", Parents: []string{"branchname-merge"}, ExtraInfo: "(HEAD -> main, origin/main)"},
		{Hash: "branchname-merge", Parents: []string{"branchname-base", "branchname-feature"}},
		{Hash: "branchname-feature", Parents: []string{"branchname-base"}, ExtraInfo: "(tag: v1)"},
		{Hash: "branchname-base", Parents: []string{}},
	}
	tests := []struct {
		name       string
		graphStyle string
		index      int
		expected   string
	}{
		{name: "commit on the checked-out branch", graphStyle: "lanes", index: 4, expected: "main"},
		{name: "branch that was deleted after it was merged, with just a tag", graphStyle: "lanes", index: 3, expected: "main"},
		{name: "rebase todo", graphStyle: "lanes", index: 0, expected: ""},
		{name: "classic graph", graphStyle: "classic", index: 4, expected: ""},
	}

	common := common.NewDummyCommon()
	branches := []*models.Branch{{Name: "main", CommitHash: "branchname-main"}}
	hashPool := &utils.StringPool{}
	commits := lo.Map(commitOpts, func(opts models.NewCommitOpts, _ int) *models.Commit { return models.NewCommit(hashPool, opts) })
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			common.UserConfig().Git.Log.GraphStyle = test.graphStyle

			assert.Equal(t, test.expected, BranchNameOfCommit(common, commits, branches, test.index))
		})
	}
}
