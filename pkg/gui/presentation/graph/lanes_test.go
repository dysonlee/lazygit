package graph

import (
	"testing"

	"github.com/gookit/color"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/xo/terminfo"
)

func TestRenderLaneGraph(t *testing.T) {
	tests := []struct {
		name           string
		commitOpts     []models.NewCommitOpts
		selectedHash   string
		labelled       []string
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
			a ●
			c │  ●
			e │  │  ●
			d │  ●  │
			b ●──╯  │
			g ●     │
			f │     ●
			h ●─────╯`,
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
			x ●
			y │  ●
			m •──┤
			q │  ●
			p ●──╯`,
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
			a ●
			b │  ●
			c │  │  ●
			m •──┼──╯
			q │  ●
			p ●──╯`,
		},
		{
			name: "a merge edge joins a lane that is already waiting for the parent",
			commitOpts: []models.NewCommitOpts{
				{Hash: "a", Parents: []string{"p"}},
				{Hash: "m", Parents: []string{"q", "p"}},
				{Hash: "q", Parents: []string{"p"}},
				{Hash: "p", Parents: []string{"r"}},
			},
			expectedOutput: `
			a ●
			m ├──•
			q │  ●
			p ●──╯`,
		},
		{
			name: "a selected merge commit keeps the lane that ends where its merge edge starts",
			commitOpts: []models.NewCommitOpts{
				{Hash: "x", Parents: []string{"m"}},
				{Hash: "y", Parents: []string{"m"}},
				{Hash: "m", Parents: []string{"p", "q"}},
				{Hash: "q", Parents: []string{"p"}},
				{Hash: "p", Parents: []string{"r"}},
			},
			selectedHash: "m",
			expectedOutput: `
			x ●
			y │  ●
			m •──┤
			q │  ●
			p ●──╯`,
		},
		{
			name: "a selected merge commit keeps the lane its merge edge joins",
			commitOpts: []models.NewCommitOpts{
				{Hash: "a", Parents: []string{"p"}},
				{Hash: "m", Parents: []string{"q", "p"}},
				{Hash: "q", Parents: []string{"p"}},
				{Hash: "p", Parents: []string{"r"}},
			},
			selectedHash: "m",
			expectedOutput: `
			a ●
			m ├──•
			q │  ●
			p ●──╯`,
		},
		{
			name: "a line connects the label of a commit to its node, through the lanes to its left",
			commitOpts: []models.NewCommitOpts{
				{Hash: "a", Parents: []string{"b"}},
				{Hash: "c", Parents: []string{"b"}},
				{Hash: "b", Parents: []string{"d"}},
			},
			labelled: []string{"c"},
			expectedOutput: `
			a ●
			c │──●
			b ●──╯`,
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
			m •──╮
			c │  │  ●
			x │  ╰──●
			p ●─────╯`,
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
			a ●
			o │  ●
			n │  ●
			b ●──╯`,
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
			1 ●
			2 •──╮
			3 •──┤
			5 •──│──╮
			7 •──│──│──╮
			4 ●──╯  │  │
			B ●     │  │
			C ●     │  │`,
		},
	}

	oldColorLevel := color.ForceSetColorLevel(terminfo.ColorLevelMillions)
	defer color.ForceSetColorLevel(oldColorLevel)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			hashPool := &utils.StringPool{}

			getStyle := func(int, *models.Commit) *style.TextStyle { return &style.FgDefault }
			commits := lo.Map(test.commitOpts,
				func(opts models.NewCommitOpts, _ int) *models.Commit { return models.NewCommit(hashPool, opts) })
			pipeSets := GetLanePipeSets(commits, nil, getStyle)
			hasLabel := func(commit *models.Commit) bool { return lo.Contains(test.labelled, commit.Hash()) }
			lines := RenderAux(pipeSets, commits, hashPool.Add(lo.CoalesceOrEmpty(test.selectedHash, "blah")), LaneGlyphs, hasLabel)

			assertGraphOutput(t, test.expectedOutput, test.commitOpts, lines)
		})
	}
}

func BenchmarkRenderLaneGraph(b *testing.B) {
	oldColorLevel := color.ForceSetColorLevel(terminfo.ColorLevelMillions)
	defer color.ForceSetColorLevel(oldColorLevel)

	hashPool := &utils.StringPool{}

	commits := generateCommits(hashPool, 50)
	getStyle := func(color int, _ *models.Commit) *style.TextStyle {
		return LaneStyle(color)
	}
	b.ResetTimer()
	for b.Loop() {
		RenderAux(GetLanePipeSets(commits, nil, getStyle), commits, hashPool.Add("selected"), LaneGlyphs, nil)
	}
}

func TestLaneGraphColors(t *testing.T) {
	tests := []struct {
		name           string
		commitOpts     []models.NewCommitOpts
		expectedColors []int
	}{
		{
			name: "every branch gets the next color, even when it reuses the column of the previous one",
			commitOpts: []models.NewCommitOpts{
				{Hash: "m1", Parents: []string{"m2", "f1"}},
				{Hash: "f1", Parents: []string{"m2"}},
				{Hash: "m2", Parents: []string{"m3", "f2"}},
				{Hash: "f2", Parents: []string{"m3"}},
				{Hash: "m3", Parents: []string{"x", "f3"}},
				{Hash: "f3", Parents: []string{"x"}},
			},
			expectedColors: []int{0, 1, 0, 2, 0, 3},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			hashPool := &utils.StringPool{}
			commits := lo.Map(test.commitOpts,
				func(opts models.NewCommitOpts, _ int) *models.Commit { return models.NewCommit(hashPool, opts) })

			colorStyles := lo.Times(LaneColorCount, func(int) *style.TextStyle { return &style.TextStyle{} })
			getStyle := func(color int, c *models.Commit) *style.TextStyle { return colorStyles[color] }
			pipeSets := GetLanePipeSets(commits, nil, getStyle)

			// The color of a commit is the color of the pipe starting at it
			// in its own lane.
			colors := lo.Map(commits, func(commit *models.Commit, i int) int {
				pipe, _ := lo.Find(pipeSets[i], func(pipe Pipe) bool {
					return pipe.kind == STARTS && pipe.fromHash == commit.HashPtr() && pipe.fromPos == pipe.toPos
				})
				return lo.IndexOf(colorStyles, pipe.style)
			})

			assert.Equal(t, test.expectedColors, colors)
		})
	}
}

func TestBranchTipIndex(t *testing.T) {
	commitOpts := []models.NewCommitOpts{
		{Hash: "main", Parents: []string{"m"}},
		{Hash: "m", Parents: []string{"p", "f2"}},
		{Hash: "f2", Parents: []string{"f1"}},
		{Hash: "f1", Parents: []string{"p"}},
		{Hash: "other", Parents: []string{"p"}},
		{Hash: "loose", Parents: []string{"p"}},
		{Hash: "p", Parents: []string{"q"}},
	}
	branchTips := []string{"main", "other"}
	tests := []struct {
		selected    string
		expectedTip string
	}{
		{selected: "main", expectedTip: "main"},
		{selected: "m", expectedTip: "main"},
		{selected: "p", expectedTip: "main"},
		{selected: "other", expectedTip: "other"},
		// f2 has no branch anymore, so its chain continues at the merge that
		// it was merged in with
		{selected: "f2", expectedTip: "main"},
		{selected: "f1", expectedTip: "main"},
		// neither a branch nor a merge to follow
		{selected: "loose", expectedTip: "loose"},
	}

	hashPool := &utils.StringPool{}
	commits := lo.Map(commitOpts, func(opts models.NewCommitOpts, _ int) *models.Commit { return models.NewCommit(hashPool, opts) })
	pipeSets := GetLanePipeSets(commits, nil, func(int, *models.Commit) *style.TextStyle { return &style.FgDefault })
	hasBranch := func(commit *models.Commit) bool { return lo.Contains(branchTips, commit.Hash()) }
	for _, test := range tests {
		t.Run(test.selected, func(t *testing.T) {
			_, selectedIdx, _ := lo.FindIndexOf(commits, func(c *models.Commit) bool { return c.Hash() == test.selected })

			tipIdx := BranchTipIndex(commits, pipeSets, selectedIdx, hasBranch)

			assert.Equal(t, test.expectedTip, commits[tipIdx].Hash())
		})
	}
}

func TestRenderLaneGraphWithBranchDrawingGlyphs(t *testing.T) {
	tests := []struct {
		name           string
		commitOpts     []models.NewCommitOpts
		expectedOutput string
	}{
		{
			name: "a lane that ends where a merge edge starts",
			commitOpts: []models.NewCommitOpts{
				{Hash: "x", Parents: []string{"m"}},
				{Hash: "y", Parents: []string{"m"}},
				{Hash: "m", Parents: []string{"p", "q"}},
				{Hash: "q", Parents: []string{"p"}},
				{Hash: "p", Parents: []string{"r"}},
			},
			expectedOutput: `
			x ●
			y │  ●
			m •──` + "\uf5df" + `
			q │  ●
			p ●──╯`,
		},
		{
			name: "the same with a line passing through, and a merge edge joining a lane",
			commitOpts: []models.NewCommitOpts{
				{Hash: "a", Parents: []string{"m"}},
				{Hash: "b", Parents: []string{"m"}},
				{Hash: "c", Parents: []string{"m"}},
				{Hash: "m", Parents: []string{"p", "q"}},
				{Hash: "j", Parents: []string{"r", "p"}},
				{Hash: "q", Parents: []string{"p"}},
				{Hash: "p", Parents: []string{"s"}},
			},
			expectedOutput: `
			a ●
			b │  ●
			c │  │  ●
			m •──` + "\uf5e8" + `──╯
			j ` + "\uf5dc" + `──│──•
			q │  ●  │
			p ●──╯  │`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			hashPool := &utils.StringPool{}
			commits := lo.Map(test.commitOpts, func(opts models.NewCommitOpts, _ int) *models.Commit { return models.NewCommit(hashPool, opts) })
			pipeSets := GetLanePipeSets(commits, nil, func(int, *models.Commit) *style.TextStyle { return &style.FgDefault })
			lines := RenderAux(pipeSets, commits, hashPool.Add("blah"), WithBranchDrawingGlyphs(LaneGlyphs), nil)

			assertGraphOutput(t, test.expectedOutput, test.commitOpts, lines)
		})
	}
}

func TestRenderLaneGraphKeepsTheMainBranchInItsLane(t *testing.T) {
	commitOpts := []models.NewCommitOpts{
		{Hash: "d", Parents: []string{"x"}},
		{Hash: "m", Parents: []string{"x"}},
		{Hash: "x", Parents: []string{"y"}},
		{Hash: "y", Parents: []string{"z"}},
	}

	hashPool := &utils.StringPool{}
	commits := lo.Map(commitOpts, func(opts models.NewCommitOpts, _ int) *models.Commit { return models.NewCommit(hashPool, opts) })
	isMainBranchTip := func(commit *models.Commit) bool { return commit.Hash() == "m" }
	pipeSets := GetLanePipeSets(commits, isMainBranchTip, func(int, *models.Commit) *style.TextStyle { return &style.FgDefault })
	lines := RenderAux(pipeSets, commits, hashPool.Add("blah"), LaneGlyphs, nil)

	// x is on both branches, and continues in the lane of the main branch
	assertGraphOutput(t, `
		d ●
		m │  ●
		x ╰──●
		y    ●`, commitOpts, lines)
}

func TestRenderLaneGraphOpensMergedBranchesInTheLeftmostFreeLane(t *testing.T) {
	commitOpts := []models.NewCommitOpts{
		{Hash: "d", Parents: []string{"x"}},
		{Hash: "m", Parents: []string{"x"}},
		{Hash: "x", Parents: []string{"w", "r"}},
		{Hash: "r", Parents: []string{"w"}},
		{Hash: "w", Parents: []string{"v"}},
	}

	hashPool := &utils.StringPool{}
	commits := lo.Map(commitOpts, func(opts models.NewCommitOpts, _ int) *models.Commit { return models.NewCommit(hashPool, opts) })
	isMainBranchTip := func(commit *models.Commit) bool { return commit.Hash() == "m" }
	pipeSets := GetLanePipeSets(commits, isMainBranchTip, func(int, *models.Commit) *style.TextStyle { return &style.FgDefault })
	lines := RenderAux(pipeSets, commits, hashPool.Add("blah"), LaneGlyphs, nil)

	// The lane of d ends at x, in the main branch's lane, and the branch that
	// x merges takes the lane that d left free, to the left of x
	assertGraphOutput(t, `
		d ●
		m │  ●
		x ├──•
		r ●  │
		w ╰──●`, commitOpts, lines)
}
