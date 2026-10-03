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
	// whether the edge is part of the chain of first parents of a main branch
	mainBranch bool
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
//
// The commits that isMainBranchTip (which may be nil) returns true for are the
// tips of main branches, like master. Where a main branch and another branch
// share their history, the shared commits continue in the main branch's lane,
// so that the main branch's history stays in one line.
func GetLanePipeSets(
	commits []*models.Commit,
	isMainBranchTip func(*models.Commit) bool,
	getStyle func(color int, c *models.Commit) *style.TextStyle,
) [][]Pipe {
	layout := laneLayout{}

	return lo.Map(commits, func(commit *models.Commit, _ int) []Pipe {
		var pipes []Pipe
		isMainBranch := isMainBranchTip != nil && isMainBranchTip(commit)
		layout, pipes = layout.next(commit, isMainBranch, getStyle)
		return pipes
	})
}

func (self laneLayout) next(commit *models.Commit, isMainBranchTip bool, getStyle func(color int, c *models.Commit) *style.TextStyle) (laneLayout, []Pipe) {
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
	isMainBranch := isMainBranchTip || (lanes[pos].mainBranch && equalHashes(lanes[pos].toHash, commit.HashPtr()))
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
			mainBranch:  isMainBranch,
		}
		pipes = append(pipes, lanes[pos].pipe(pos, pos, STARTS))
	}

	for _, parentHash := range lo.Drop(commit.ParentPtrs(), 1) {
		// If another lane already leads to the parent, the merge edge joins it
		// rather than running alongside it in a lane of its own.
		if joinLane := slices.IndexFunc(lanes, func(l lane) bool { return equalHashes(l.toHash, parentHash) }); joinLane != -1 && joinLane != pos {
			edge := lane{fromHash: commit.HashPtr(), toHash: parentHash, style: lanes[joinLane].style}
			pipes = append(pipes, edge.pipe(pos, joinLane, STARTS))
			continue
		}

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
// one straight line, with merge edges bending into it; and among those, the
// lane of a main branch keeps the main branch in one line. If no lane is waiting
// for the commit, it is a branch tip and takes the leftmost free lane (which
// may be a new one at the right edge).
func commitLane(lanes []lane, commit *models.Commit) int {
	isWaitingForCommit := func(l lane) bool { return equalHashes(l.toHash, commit.HashPtr()) }
	if i := slices.IndexFunc(lanes, func(l lane) bool { return isWaitingForCommit(l) && l.firstParent && l.mainBranch }); i != -1 {
		return i
	}
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

// BranchTipIndex returns the index of the commit whose branch the commit at
// index i is on, as far as the lane graph shows it: the first commit with a
// branch (as told by hasBranch) up the chain of first parents that the commit
// continues in its lane. If the top of that chain has no branch, e.g. because
// the branch was deleted after it was merged, the search goes on from the
// merge commit that it was merged with. Without any branch on the way, it
// returns the last commit that it got to.
func BranchTipIndex(commits []*models.Commit, pipeSets [][]Pipe, i int, hasBranch func(*models.Commit) bool) int {
	for !hasBranch(commits[i]) {
		next := firstParentChildIndex(commits, pipeSets, i)
		if next == -1 {
			next = mergeChildIndex(commits, i)
		}
		if next == -1 {
			return i
		}
		i = next
	}
	return i
}

// mergeChildIndex returns the index of the nearest merge commit above the
// commit at index i that merged it, i.e. that has it as one of its non-first
// parents, or -1 if there is none.
func mergeChildIndex(commits []*models.Commit, i int) int {
	for j := i - 1; j >= 0; j-- {
		if lo.Contains(lo.Drop(commits[j].ParentPtrs(), 1), commits[i].HashPtr()) {
			return j
		}
	}
	return -1
}

// firstParentChildIndex returns the index of the commit whose edge to its first
// parent comes down the lane into the commit at index i, or -1 if there is none.
func firstParentChildIndex(commits []*models.Commit, pipeSets [][]Pipe, i int) int {
	pos := CommitPos(pipeSets[i])
	incoming, found := lo.Find(pipeSets[i], func(pipe Pipe) bool {
		return pipe.kind == TERMINATES && pipe.fromPos == pos && pipe.toPos == pos
	})
	if !found {
		return -1
	}

	for j := i - 1; j >= 0; j-- {
		if commits[j].HashPtr() == incoming.fromHash {
			if len(commits[j].ParentPtrs()) > 0 && commits[j].ParentPtrs()[0] == commits[i].HashPtr() {
				return j
			}
			return -1
		}
	}
	return -1
}
