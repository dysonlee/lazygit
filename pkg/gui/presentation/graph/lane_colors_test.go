package graph

import (
	"testing"

	"github.com/jesseduffield/generics/set"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func TestPickLaneColor(t *testing.T) {
	last := LaneColorCount - 1
	tests := []struct {
		name              string
		next              int
		inUse             []int
		expectedColor     int
		expectedNextColor int
	}{
		{name: "takes the next color if it is unused", next: 3, inUse: []int{0, 1}, expectedColor: 3, expectedNextColor: 4},
		{name: "skips colors in use", next: 0, inUse: []int{0, 1}, expectedColor: 2, expectedNextColor: 3},
		{name: "wraps around", next: last, inUse: []int{last}, expectedColor: 0, expectedNextColor: 1},
		{name: "takes the next color if all are in use", next: 4, inUse: lo.Range(LaneColorCount), expectedColor: 4, expectedNextColor: 5},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			inUse := set.NewFromSlice(test.inUse)
			color, nextColor := pickLaneColor(test.next, inUse.Includes)

			assert.Equal(t, test.expectedColor, color)
			assert.Equal(t, test.expectedNextColor, nextColor)
		})
	}
}

func TestLaneLabelStyle(t *testing.T) {
	hashPool := &utils.StringPool{}
	commits := lo.Map([]models.NewCommitOpts{
		{Hash: "a", Parents: []string{"c"}},
		{Hash: "b", Parents: []string{"c"}},
		{Hash: "c", Parents: []string{}},
	}, func(opts models.NewCommitOpts, _ int) *models.Commit { return models.NewCommit(hashPool, opts) })
	dimmedHash := commits[1].HashPtr()
	pipeSets := GetLanePipeSets(commits, func(color int, commit *models.Commit) *style.TextStyle {
		if commit.HashPtr() == dimmedHash {
			return DimmedLaneStyle(color)
		}
		return LaneStyle(color)
	})

	assert.Equal(t, laneLabelStyles[0], LaneLabelStyle(CommitStyle(pipeSets[0])))
	assert.Equal(t, laneLabelStyles[1], LaneLabelStyle(CommitStyle(pipeSets[1])), "a dimmed commit's label is not dimmed")
	assert.Equal(t, laneLabelStyles[0], LaneLabelStyle(CommitStyle(pipeSets[2])))

	classicPipeSets := GetPipeSets(commits, func(*models.Commit) *style.TextStyle { return &style.FgDefault })
	assert.Nil(t, LaneLabelStyle(CommitStyle(classicPipeSets[0])))
}
