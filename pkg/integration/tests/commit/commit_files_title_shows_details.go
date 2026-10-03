package commit

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var CommitFilesTitleShowsDetails = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "The title of a commit's files shows its author, date, and the branch it is on in the lane graph",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(config *config.AppConfig) {
		config.GetUserConfig().Git.Log.GraphStyle = "lanes"
		config.GetUserConfig().Gui.TimeFormat = "2006-01-02"
	},
	SetupRepo: func(shell *Shell) {
		shell.SetAuthor("Jane Doe", "jane@example.com")
		shell.EmptyCommitWithDate("one", "2024-03-01T10:00:00")
		shell.NewBranch("feature")
		shell.EmptyCommitWithDate("two", "2024-03-02T10:00:00")
		shell.EmptyCommitWithDate("three", "2024-03-03T10:00:00")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().
			Focus().
			NavigateToLine(Contains("two")).
			PressEnter()

		t.Views().CommitFiles().
			IsFocused().
			Title(Contains("two) · Jane Doe · 2024-03-02 · feature"))
	},
})
