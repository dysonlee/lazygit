package context

import (
	"fmt"
	"strings"
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/filetree"
	"github.com/jesseduffield/lazygit/pkg/gui/presentation"
	"github.com/jesseduffield/lazygit/pkg/gui/presentation/icons"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
)

type CommitFilesContext struct {
	*filetree.CommitFileTreeViewModel
	*ListContextTrait
	*DynamicTitleBuilder

	// shown after the title, e.g. the author and date of the commit
	titleDetails []string
}

var (
	_ types.IListContext       = (*CommitFilesContext)(nil)
	_ types.DiffableContext    = (*CommitFilesContext)(nil)
	_ types.IFilterableContext = (*CommitFilesContext)(nil)
)

func NewCommitFilesContext(c *ContextCommon) *CommitFilesContext {
	viewModel := filetree.NewCommitFileTreeViewModel(
		func() []*models.CommitFile { return c.Model().CommitFiles },
		c.Common,
		c.UserConfig().Gui.ShowFileTree,
	)

	getDisplayStrings := func(_ int, _ int) [][]string {
		if viewModel.Len() == 0 {
			return [][]string{{style.FgRed.Sprint("(none)")}}
		}

		showFileIcons := icons.IsIconEnabled() && c.UserConfig().Gui.ShowFileIcons
		lines := presentation.RenderCommitFileTree(viewModel, c.Git().Patch.PatchBuilder, showFileIcons, &c.UserConfig().Gui.CustomIcons)
		return lo.Map(lines, func(line string, _ int) []string {
			return []string{line}
		})
	}

	ctx := &CommitFilesContext{
		CommitFileTreeViewModel: viewModel,
		DynamicTitleBuilder:     NewDynamicTitleBuilder(c.Tr.CommitFilesDynamicTitle),
		ListContextTrait: &ListContextTrait{
			Context: NewSimpleContext(
				NewBaseContext(NewBaseContextOpts{
					View:       c.Views().CommitFiles,
					WindowName: "commits",
					Key:        COMMIT_FILES_CONTEXT_KEY,
					Kind:       types.SIDE_CONTEXT,
					Focusable:  true,
					Transient:  true,
				}),
			),
			ListRenderer: ListRenderer{
				list:              viewModel,
				getDisplayStrings: getDisplayStrings,
			},
			c: c,
		},
	}

	return ctx
}

func (self *CommitFilesContext) GetDiffTerminals() []string {
	return []string{self.GetRef().RefName()}
}

func (self *CommitFilesContext) RefForAdjustingLineNumberInDiff() string {
	if refs := self.GetRefRange(); refs != nil {
		return refs.To.RefName()
	}
	return self.GetRef().RefName()
}

func (self *CommitFilesContext) GetFromAndToForDiff() (string, string) {
	if refs := self.GetRefRange(); refs != nil {
		return refs.From.ParentRefName(), refs.To.RefName()
	}
	ref := self.GetRef()
	return ref.ParentRefName(), ref.RefName()
}

// ReInit shows the files of the given ref or range. The title shows the
// author and date of a single commit, and the branch it is on, if known, as
// the commits list may leave these out to make room for the graph.
func (self *CommitFilesContext) ReInit(ref models.Ref, refRange *types.RefRange, branchName string) {
	self.SetRef(ref)
	self.SetRefRange(refRange)
	if refRange != nil {
		self.SetTitleRef(fmt.Sprintf("%s-%s", refRange.From.ShortRefName(), refRange.To.ShortRefName()))
		self.titleDetails = nil
		self.GetView().Title = self.Title()
		return
	}

	self.SetTitleRef(ref.Description())
	details := []string{}
	if commit, ok := ref.(*models.Commit); ok && !commit.IsTODO() {
		userConfig := self.c.UserConfig()
		details = append(details,
			commit.AuthorName,
			utils.UnixToDateSmart(time.Now(), commit.UnixTimestamp, userConfig.Gui.TimeFormat, userConfig.Gui.ShortTimeFormat))
	}
	self.titleDetails = lo.Compact(append(details, branchName))
	self.GetView().Title = self.Title()
}

func (self *CommitFilesContext) Title() string {
	return strings.Join(append([]string{self.DynamicTitleBuilder.Title()}, self.titleDetails...), " · ")
}
