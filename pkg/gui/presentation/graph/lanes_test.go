package graph

import (
	"testing"

	"github.com/gookit/color"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/presentation/authors"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
	"github.com/xo/terminfo"
)

func TestRenderLaneGraph(t *testing.T) {
	tests := []struct {
		name           string
		commitOpts     []models.NewCommitOpts
		expectedOutput string
	}{
		{
			name: "lanes keep their column when a lane to their left ends",
			commitOpts: []models.NewCommitOpts{
				{Hash: "a", Parents: []string{"b"}},
				{Hash: "c", Parents: []string{"d"}},
				{Hash: "e", Parents: []string{"f"}},
				{Hash: "d", Parents: []string{"b"}},
				{Hash: "b", Parents: []string{"g"}},
				{Hash: "g", Parents: []string{"h"}},
				{Hash: "f", Parents: []string{"h"}},
				{Hash: "h", Parents: []string{"i"}},
			},
			expectedOutput: `
			a ○
			c │ ○
			e │ │ ○
			d │ ○ │
			b ○─╯ │
			g ○   │
			f │   ○
			h ○───╯`,
		},
		{
			name: "a merge parent reuses a lane that ends on the same row, drawn as a junction",
			commitOpts: []models.NewCommitOpts{
				{Hash: "x", Parents: []string{"m"}},
				{Hash: "y", Parents: []string{"m"}},
				{Hash: "m", Parents: []string{"p", "q"}},
				{Hash: "q", Parents: []string{"p"}},
				{Hash: "p", Parents: []string{"r"}},
			},
			expectedOutput: `
			x ○
			y │ ○
			m ◎─┤
			q │ ○
			p ○─╯`,
		},
		{
			name: "a junction that a horizontal line passes through is drawn as a cross",
			commitOpts: []models.NewCommitOpts{
				{Hash: "a", Parents: []string{"m"}},
				{Hash: "b", Parents: []string{"m"}},
				{Hash: "c", Parents: []string{"m"}},
				{Hash: "m", Parents: []string{"p", "q"}},
				{Hash: "q", Parents: []string{"p"}},
				{Hash: "p", Parents: []string{"r"}},
			},
			expectedOutput: `
			a ○
			b │ ○
			c │ │ ○
			m ◎─┼─╯
			q │ ○
			p ○─╯`,
		},
		{
			name: "a commit stays in the lane of its first-parent chain rather than the leftmost lane waiting for it",
			commitOpts: []models.NewCommitOpts{
				{Hash: "m", Parents: []string{"p", "x"}},
				{Hash: "c", Parents: []string{"x"}},
				{Hash: "x", Parents: []string{"p"}},
				{Hash: "p", Parents: []string{"q"}},
			},
			expectedOutput: `
			m ◎─╮
			c │ │ ○
			x │ ╰─○
			p ○───╯`,
		},
		{
			name: "a root commit frees its lane for the next branch tip",
			commitOpts: []models.NewCommitOpts{
				{Hash: "a", Parents: []string{"b"}},
				{Hash: "o", Parents: []string{}},
				{Hash: "n", Parents: []string{"b"}},
				{Hash: "b", Parents: []string{}},
			},
			expectedOutput: `
			a ○
			o │ ○
			n │ ○
			b ○─╯`,
		},
		{
			name: "lanes don't move left into the space left by converging lanes",
			commitOpts: []models.NewCommitOpts{
				{Hash: "1", Parents: []string{"2"}},
				{Hash: "2", Parents: []string{"3", "4"}},
				{Hash: "3", Parents: []string{"5", "4"}},
				{Hash: "5", Parents: []string{"7", "8"}},
				{Hash: "7", Parents: []string{"4", "A"}},
				{Hash: "4", Parents: []string{"B"}},
				{Hash: "B", Parents: []string{"C"}},
				{Hash: "C", Parents: []string{"D"}},
			},
			expectedOutput: `
			1 ○
			2 ◎─╮
			3 ◎─│─╮
			5 ◎─│─│─╮
			7 ◎─│─│─│─╮
			4 ○─┴─╯ │ │
			B ○     │ │
			C ○     │ │`,
		},
	}

	oldColorLevel := color.ForceSetColorLevel(terminfo.ColorLevelMillions)
	defer color.ForceSetColorLevel(oldColorLevel)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			hashPool := &utils.StringPool{}

			getStyle := func(c *models.Commit) *style.TextStyle { return &style.FgDefault }
			commits := lo.Map(test.commitOpts,
				func(opts models.NewCommitOpts, _ int) *models.Commit { return models.NewCommit(hashPool, opts) })
			pipeSets := GetLanePipeSets(commits, getStyle)
			lines := RenderAux(pipeSets, commits, hashPool.Add("blah"), LaneGlyphs)

			assertGraphOutput(t, test.expectedOutput, test.commitOpts, lines)
		})
	}
}

func BenchmarkRenderLaneGraph(b *testing.B) {
	oldColorLevel := color.ForceSetColorLevel(terminfo.ColorLevelMillions)
	defer color.ForceSetColorLevel(oldColorLevel)

	hashPool := &utils.StringPool{}

	commits := generateCommits(hashPool, 50)
	getStyle := func(commit *models.Commit) *style.TextStyle {
		return authors.AuthorStyle(commit.AuthorName)
	}
	b.ResetTimer()
	for b.Loop() {
		RenderAux(GetLanePipeSets(commits, getStyle), commits, hashPool.Add("selected"), LaneGlyphs)
	}
}
