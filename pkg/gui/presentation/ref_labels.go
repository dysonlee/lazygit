package presentation

import (
	"fmt"
	"slices"
	"strings"

	"github.com/jesseduffield/generics/set"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
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
// number of the others, and then a line that continues into the line to the
// commit's node in the graph, filling the given width. A name too long for the
// width is truncated, leaving room for at least some of the line, which shows
// which commit the label belongs to. Without labels, it renders blanks.
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
	pill := prefix + utils.TruncateWithEllipsis(label.name, nameWidth) + suffix
	line := strings.Repeat("─", max(width-utils.StringWidth(pill), 0))

	return labelStyle.Sprint(pill) + lineStyle.Sprint(line)
}

// The width of the labels in front of the lane graph
const refLabelsWidth = 20

// The labels of the commits of the rows of the graph that are rendered
type refLabelColumn struct {
	labels   [][]refLabel
	labelled *set.Set[*string]
}

func newRefLabelColumn(commits []*models.Commit, branches []*models.Branch) refLabelColumn {
	localBranchNames := set.NewFromSlice(lo.Map(branches, func(branch *models.Branch, _ int) string { return branch.Name }))
	labels := lo.Map(commits, func(commit *models.Commit, _ int) []refLabel {
		return commitRefLabels(commit, localBranchNames)
	})
	labelled := set.NewFromSlice(lo.FilterMap(commits, func(commit *models.Commit, i int) (*string, bool) {
		return commit.HashPtr(), len(labels[i]) > 0
	}))
	return refLabelColumn{labels: labels, labelled: labelled}
}

func (self refLabelColumn) hasLabel(commit *models.Commit) bool {
	return self.labelled.Includes(commit.HashPtr())
}

// prependTo puts the labels in front of the graph lines, colored like the
// commits they belong to.
func (self refLabelColumn) prependTo(graphLines []string, pipeSets [][]graph.Pipe, withIcons bool) []string {
	return lo.Map(graphLines, func(graphLine string, i int) string {
		commitStyle := graph.CommitStyle(pipeSets[i])
		labels := renderRefLabels(self.labels[i], refLabelsWidth, graph.LaneLabelStyle(commitStyle), commitStyle, withIcons)
		return labels + graphLine
	})
}
