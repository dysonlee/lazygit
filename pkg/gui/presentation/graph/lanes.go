package graph

import (
	"slices"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/samber/lo"
)

// A lane is a column of the graph that carries an edge from a commit down to
// its parent. Unlike the pipes of the classic layout, a lane never moves to
// another column, which keeps every branch in a straight vertical line.
type lane struct {
	fromHash *string
	// nil when the lane is free
	toHash *string
	color  int
	style  *style.TextStyle
	// whether the edge goes from a commit to its first parent, as opposed to
	// from a merge commit to one of its other parents
	firstParent bool
}

func (self lane) isFree() bool {
	return self.toHash == nil
}

func (self lane) pipe(fromPos, toPos int, kind PipeKind) Pipe {
	return Pipe{
		fromPos:  int16(fromPos),
		toPos:    int16(toPos),
		fromHash: self.fromHash,
		toHash:   self.toHash,
		kind:     kind,
		style:    self.style,
	}
}

// The state of the lane layout after a row of the graph
type laneLayout struct {
	lanes []lane
	// the color to try first for the next branch
	nextColor int
}

// GetLanePipeSets lays out the graph in fixed lanes and returns, for each
// commit, the pipes that RenderAux draws for its row. Every branch gets its
// own color: a commit takes the color of the lane it continues, and each new
// branch tip or merged branch gets the next lane color in turn. getStyle
// returns the style for one of these colors (in the range [0,
// LaneColorCount)) on the edges starting at the given commit.
func GetLanePipeSets(commits []*models.Commit, getStyle func(color int, c *models.Commit) *style.TextStyle) [][]Pipe {
	layout := laneLayout{}

	return lo.Map(commits, func(commit *models.Commit, _ int) []Pipe {
		var pipes []Pipe
		layout, pipes = layout.next(commit, getStyle)
		return pipes
	})
}

func (self laneLayout) next(commit *models.Commit, getStyle func(color int, c *models.Commit) *style.TextStyle) (laneLayout, []Pipe) {
	lanes := slices.Clone(self.lanes)
	nextColor := self.nextColor
	// A new branch avoids the colors of the branches on this row and the
	// previous one, so that it can be told apart from a branch that ended
	// in the same column.
	newBranchColor := func() int {
		var color int
		color, nextColor = pickLaneColor(nextColor, func(color int) bool {
			return lanesUseColor(self.lanes, color) || lanesUseColor(lanes, color)
		})
		return color
	}

	pos := commitLane(lanes, commit)
	if pos == len(lanes) {
		lanes = append(lanes, lane{})
	}
	var commitColor int
	if equalHashes(lanes[pos].toHash, commit.HashPtr()) {
		commitColor = lanes[pos].color
	} else {
		commitColor = newBranchColor()
	}

	pipes := make([]Pipe, 0, len(lanes)+len(commit.ParentPtrs()))
	for i, l := range lanes {
		if l.isFree() {
			continue
		}

		if equalHashes(l.toHash, commit.HashPtr()) {
			pipes = append(pipes, l.pipe(i, pos, TERMINATES))
			lanes[i] = lane{}
		} else {
			pipes = append(pipes, l.pipe(i, i, CONTINUES))
		}
	}

	if commit.IsFirstCommit() {
		// renderPipeSet finds a commit's position from the pipe starting at
		// it, so a root commit gets one too, but its lane stays free.
		edge := lane{fromHash: commit.HashPtr(), toHash: &EmptyTreeCommitHash, style: getStyle(commitColor, commit)}
		pipes = append(pipes, edge.pipe(pos, pos, STARTS))
	} else {
		lanes[pos] = lane{
			fromHash:    commit.HashPtr(),
			toHash:      commit.ParentPtrs()[0],
			color:       commitColor,
			style:       getStyle(commitColor, commit),
			firstParent: true,
		}
		pipes = append(pipes, lanes[pos].pipe(pos, pos, STARTS))
	}

	for _, parentHash := range lo.Drop(commit.ParentPtrs(), 1) {
		mergeLane := mergeParentLane(lanes, pos)
		if mergeLane == len(lanes) {
			lanes = append(lanes, lane{})
		}
		color := newBranchColor()
		lanes[mergeLane] = lane{fromHash: commit.HashPtr(), toHash: parentHash, color: color, style: getStyle(color, commit)}
		pipes = append(pipes, lanes[mergeLane].pipe(pos, mergeLane, STARTS))
	}

	for len(lanes) > 0 && lanes[len(lanes)-1].isFree() {
		lanes = lanes[:len(lanes)-1]
	}

	sortPipes(pipes)

	return laneLayout{lanes: lanes, nextColor: nextColor}, pipes
}

func lanesUseColor(lanes []lane, color int) bool {
	return lo.ContainsBy(lanes, func(l lane) bool { return !l.isFree() && l.color == color })
}

// commitLane returns the lane a commit is drawn in. Preferring a lane that
// comes from a child's first-parent edge keeps a branch's chain of commits in
// one straight line, with merge edges bending into it. If no lane is waiting
// for the commit, it is a branch tip and takes the leftmost free lane (which
// may be a new one at the right edge).
func commitLane(lanes []lane, commit *models.Commit) int {
	isWaitingForCommit := func(l lane) bool { return equalHashes(l.toHash, commit.HashPtr()) }
	if i := slices.IndexFunc(lanes, func(l lane) bool { return isWaitingForCommit(l) && l.firstParent }); i != -1 {
		return i
	}
	if i := slices.IndexFunc(lanes, isWaitingForCommit); i != -1 {
		return i
	}

	if i := slices.IndexFunc(lanes, lane.isFree); i != -1 {
		return i
	}

	return len(lanes)
}

// mergeParentLane returns the lane for the edge from a merge commit in lane
// pos to one of its non-first parents: the leftmost free lane to the right of
// the commit. This includes a lane that ends at the commit, so that a series
// of branches that were each started from the merge of the previous one
// stays in one column.
func mergeParentLane(lanes []lane, pos int) int {
	for i := pos + 1; i < len(lanes); i++ {
		if lanes[i].isFree() {
			return i
		}
	}

	return len(lanes)
}
