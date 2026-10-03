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

// GetLanePipeSets lays out the graph in fixed lanes and returns, for each
// commit, the pipes that RenderAux draws for its row.
func GetLanePipeSets(commits []*models.Commit, getStyle func(c *models.Commit) *style.TextStyle) [][]Pipe {
	lanes := []lane{}

	return lo.Map(commits, func(commit *models.Commit, _ int) []Pipe {
		var pipes []Pipe
		lanes, pipes = getNextLanePipes(lanes, commit, getStyle)
		return pipes
	})
}

func getNextLanePipes(prevLanes []lane, commit *models.Commit, getStyle func(c *models.Commit) *style.TextStyle) ([]lane, []Pipe) {
	lanes := slices.Clone(prevLanes)
	pos := commitLane(lanes, commit)
	if pos == len(lanes) {
		lanes = append(lanes, lane{})
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

	commitStyle := getStyle(commit)
	if commit.IsFirstCommit() {
		// renderPipeSet finds a commit's position from the pipe starting at
		// it, so a root commit gets one too, but its lane stays free.
		edge := lane{fromHash: commit.HashPtr(), toHash: &EmptyTreeCommitHash, style: commitStyle}
		pipes = append(pipes, edge.pipe(pos, pos, STARTS))
	} else {
		lanes[pos] = lane{fromHash: commit.HashPtr(), toHash: commit.ParentPtrs()[0], style: commitStyle, firstParent: true}
		pipes = append(pipes, lanes[pos].pipe(pos, pos, STARTS))
	}

	for _, parentHash := range lo.Drop(commit.ParentPtrs(), 1) {
		mergeLane := mergeParentLane(lanes, pos)
		if mergeLane == len(lanes) {
			lanes = append(lanes, lane{})
		}
		lanes[mergeLane] = lane{fromHash: commit.HashPtr(), toHash: parentHash, style: commitStyle}
		pipes = append(pipes, lanes[mergeLane].pipe(pos, mergeLane, STARTS))
	}

	for len(lanes) > 0 && lanes[len(lanes)-1].isFree() {
		lanes = lanes[:len(lanes)-1]
	}

	sortPipes(pipes)

	return lanes, pipes
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
