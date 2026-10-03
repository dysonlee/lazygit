package presentation

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/jesseduffield/generics/set"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/jesseduffield/lazygit/pkg/gui/presentation/graph"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
)

// A label for a ref pointing at a commit, shown in front of the lane graph
type refLabel struct {
	name string
	// the checked-out branch, or HEAD itself if it is detached
	head bool
	// a local branch, or a remote branch, or both if the local branch has a
	// remote branch of the same name at the same commit
	local  bool
	remote bool
	tag    bool
}

// commitRefLabels returns the labels for the refs that point at the commit,
// with the checked-out branch first, then the other local branches, remote
// branches, and tags. A remote branch whose name matches a local branch at the
// same commit shares its label.
func commitRefLabels(commit *models.Commit, localBranchNames *set.Set[string]) []refLabel {
	var heads, locals, remotes, tags []refLabel
	for _, decoration := range commitDecorations(commit) {
		switch {
		case decoration == "HEAD":
			heads = append(heads, refLabel{name: "HEAD", head: true})
		case strings.HasPrefix(decoration, "HEAD -> "):
			heads = append(heads, refLabel{name: strings.TrimPrefix(decoration, "HEAD -> "), head: true, local: true})
		case strings.HasPrefix(decoration, "tag: "):
			tags = append(tags, refLabel{name: strings.TrimPrefix(decoration, "tag: "), tag: true})
		case strings.HasSuffix(decoration, "/HEAD"):
			// the default branch of a remote, which is also decorated by its name
		case strings.HasPrefix(decoration, "refs/"):
			// refs that are neither branches nor tags, like refs/stash
		case localBranchNames.Includes(decoration):
			locals = append(locals, refLabel{name: decoration, local: true})
		default:
			remotes = append(remotes, refLabel{name: decoration, remote: true})
		}
	}

	branches := append(heads, locals...)
	var remotesWithoutLocal []refLabel
	for _, remote := range remotes {
		_, branchName, _ := strings.Cut(remote.name, "/")
		i := slices.IndexFunc(branches, func(branch refLabel) bool { return branch.local && branch.name == branchName })
		if i == -1 {
			remotesWithoutLocal = append(remotesWithoutLocal, remote)
		} else {
			branches[i].remote = true
		}
	}

	return lo.Flatten([][]refLabel{branches, remotesWithoutLocal, tags})
}

// Nerd Font icons for the kinds of refs in a label. They are Font Awesome
// icons, which are at the same code points in Nerd Fonts 2 and 3.
const (
	localBranchIcon  = "" // nf-fa-laptop
	remoteBranchIcon = "" // nf-fa-cloud
	tagIcon          = "" // nf-fa-tag
)

// renderRefLabels renders the first of a commit's labels, followed by the
// number of the others, filling the given width with a line in lineStyle that
// continues into the line to the commit's node in the graph, or with blanks if
// lineStyle is nil. A name too long for the width is truncated, leaving room
// for at least some of the line, which shows which commit the label belongs
// to. Without labels, it renders blanks.
func renderRefLabels(labels []refLabel, width int, labelStyle *style.TextStyle, lineStyle *style.TextStyle, withIcons bool) string {
	if len(labels) == 0 {
		return strings.Repeat(" ", width)
	}

	label := labels[0]
	prefix := lo.Ternary(label.head, " ✓ ", " ")
	suffix := ""
	if withIcons {
		icons := lo.Compact([]string{
			lo.Ternary(label.local, localBranchIcon, ""),
			lo.Ternary(label.remote, remoteBranchIcon, ""),
			lo.Ternary(label.tag, tagIcon, ""),
		})
		suffix += lo.Ternary(len(icons) > 0, " "+strings.Join(icons, " "), "")
	}
	if len(labels) > 1 {
		suffix += fmt.Sprintf(" +%d", len(labels)-1)
	}
	suffix += " "

	const minLineWidth = 1
	nameWidth := width - minLineWidth - utils.StringWidth(prefix) - utils.StringWidth(suffix)
	pill := prefix + shortenRefName(labelName(label, withIcons), nameWidth) + suffix
	lineWidth := max(width-utils.StringWidth(pill), 0)
	if lineStyle == nil {
		return labelStyle.Sprint(pill) + strings.Repeat(" ", lineWidth)
	}
	return labelStyle.Sprint(pill) + lineStyle.Sprint(graph.LabelLine(lineWidth))
}

// labelName returns the name to show for a label. With icons, which tell
// whether a branch is a remote one, a remote branch goes without the name of
// its remote.
func labelName(label refLabel, withIcons bool) string {
	if withIcons && label.remote && !label.local {
		if _, branchName, found := strings.Cut(label.name, "/"); found {
			return branchName
		}
	}
	return label.name
}

// shortenRefName fits a ref name into the given width. It abbreviates the
// leading segments of a name like "hotfix/ROAR-13955" to their first letter,
// one after the other, since the last segment usually tells most about the
// branch, e.g. by a ticket number; if that isn't enough, it leaves out the
// middle of the name.
func shortenRefName(name string, width int) string {
	segments := strings.Split(name, "/")
	for i := 0; i < len(segments)-1 && utils.StringWidth(strings.Join(segments, "/")) > width; i++ {
		if firstLetter, _ := utf8.DecodeRuneInString(segments[i]); firstLetter != utf8.RuneError {
			segments[i] = string(firstLetter)
		}
	}
	return utils.TruncateWithEllipsisInMiddle(strings.Join(segments, "/"), width)
}

// The width of the labels in front of the lane graph
const refLabelsWidth = 18

// The labels of the commits of the rows of the graph that are rendered
type refLabelColumn struct {
	commits  []*models.Commit
	labels   [][]refLabel
	pipeSets [][]graph.Pipe
	// whether the labels of a row are those of the branch that the commit is
	// on rather than of refs pointing at the commit itself
	branchLabel []bool
	// the row of the selected commit, or -1 if it isn't visible
	selectedIdx int
}

// newRefLabelColumn returns the labels of the commits in the range
// [start, end) of the graph's commits. If the selected commit is among them
// and has no refs of its own, it gets the labels of the branch it is on, so
// that it's clear which branch it belongs to. Only the selected commit's label
// is connected to the commit's node by a line, which would otherwise be
// mistaken for the lines of the branches.
func newRefLabelColumn(
	commits []*models.Commit,
	pipeSets [][]graph.Pipe,
	start int,
	end int,
	branches []*models.Branch,
	mainBranches []string,
	selectedCommitHashPtr *string,
) refLabelColumn {
	localBranchNames := set.NewFromSlice(lo.Map(branches, func(branch *models.Branch, _ int) string { return branch.Name }))
	visibleCommits := commits[start:end]
	labels := lo.Map(visibleCommits, func(commit *models.Commit, _ int) []refLabel {
		return commitRefLabels(commit, localBranchNames)
	})
	branchLabel := make([]bool, len(visibleCommits))

	_, selectedIdx, selectedIsVisible := lo.FindIndexOf(visibleCommits, func(commit *models.Commit) bool {
		return commit.HashPtr() == selectedCommitHashPtr
	})
	if selectedIsVisible && len(labels[selectedIdx]) == 0 {
		labels[selectedIdx] = branchLabelsOfCommit(commits, pipeSets, start+selectedIdx, localBranchNames, mainBranches)
		branchLabel[selectedIdx] = true
	}

	return refLabelColumn{
		commits:     visibleCommits,
		labels:      labels,
		pipeSets:    pipeSets[start:end],
		branchLabel: branchLabel,
		selectedIdx: selectedIdx,
	}
}

// hasLineToLabel tells whether the commit's node is connected to its label in
// front of the graph by a line.
func (self refLabelColumn) hasLineToLabel(commit *models.Commit) bool {
	return self.selectedIdx != -1 && commit == self.commits[self.selectedIdx] && len(self.labels[self.selectedIdx]) > 0
}

// prependTo puts the labels in front of the graph lines, colored like the
// commits they belong to. The label of the branch of a commit shows in the
// color of the commit's lane, without a background, to tell it apart from the
// labels of refs pointing at the commit.
func (self refLabelColumn) prependTo(graphLines []string, withIcons bool) []string {
	return lo.Map(graphLines, func(graphLine string, i int) string {
		commitStyle := graph.CommitStyle(self.pipeSets[i])
		labelStyle := lo.Ternary(self.branchLabel[i], commitStyle, graph.LaneLabelStyle(commitStyle))
		lineStyle := lo.Ternary(i == self.selectedIdx, commitStyle, nil)
		labels := renderRefLabels(self.labels[i], refLabelsWidth, labelStyle, lineStyle, withIcons)
		return labels + graphLine
	})
}

// BranchNameOfCommit returns the name of the branch that the commit at the
// given index is on in the lane graph (see graph.BranchTipIndex), or an empty
// string if the commits aren't shown in the lane graph, or if there is no
// such branch.
func BranchNameOfCommit(common *common.Common, commits []*models.Commit, branches []*models.Branch, index int) string {
	mutex.Lock()
	defer mutex.Unlock()

	if common.UserConfig().Git.Log.GraphStyle != "lanes" {
		return ""
	}

	rebaseOffset := indexOfFirstNonTODOCommit(commits)
	if index < rebaseOffset || index >= len(commits) {
		return ""
	}

	graphCommits := commits[rebaseOffset:]
	pipeSets, _ := loadPipesets(graphCommits, "lanes", common.UserConfig().Git.Log.DimUnreachableCommits, common.UserConfig().Git.MainBranches)
	localBranchNames := set.NewFromSlice(lo.Map(branches, func(branch *models.Branch, _ int) string { return branch.Name }))
	labels := branchLabelsOfCommit(graphCommits, pipeSets, index-rebaseOffset, localBranchNames, common.UserConfig().Git.MainBranches)
	if len(labels) == 0 {
		return ""
	}
	return labels[0].name
}

// branchLabelsOfCommit returns the labels of the branches that the commit at
// the given index is on, leaving out tags; or nil if there is no such branch.
// These are the branches pointing at the commit, if any, or else the first
// ones up the commits that continue it, preferring a main branch (see
// graph.BranchTipIndex).
func branchLabelsOfCommit(
	commits []*models.Commit,
	pipeSets [][]graph.Pipe,
	index int,
	localBranchNames *set.Set[string],
	mainBranches []string,
) []refLabel {
	branchLabels := func(commit *models.Commit) []refLabel {
		return lo.Filter(commitRefLabels(commit, localBranchNames), func(label refLabel, _ int) bool {
			return label.local || label.remote
		})
	}

	tip := graph.BranchTipIndex(commits, pipeSets, index,
		func(commit *models.Commit) bool { return len(branchLabels(commit)) > 0 },
		func(commit *models.Commit) bool { return isMainBranchTip(commit, mainBranches) },
	)
	return branchLabels(commits[tip])
}

// isMainBranchTip tells whether one of the refs pointing at the commit is a
// local or remote main branch.
func isMainBranchTip(commit *models.Commit, mainBranches []string) bool {
	return lo.ContainsBy(commitDecorations(commit), func(decoration string) bool {
		return isMainBranchName(strings.TrimPrefix(decoration, "HEAD -> "), mainBranches)
	})
}

// isMainBranchName tells whether a local or remote branch is a main branch.
func isMainBranchName(name string, mainBranches []string) bool {
	_, remoteBranchName, isRemote := strings.Cut(name, "/")
	return lo.Contains(mainBranches, name) || (isRemote && lo.Contains(mainBranches, remoteBranchName))
}
