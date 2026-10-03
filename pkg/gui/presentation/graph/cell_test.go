package graph

import (
	"testing"

	"github.com/gookit/color"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/stretchr/testify/assert"
	"github.com/xo/terminfo"
)

func TestCachedSprintFollowsTheColorLevel(t *testing.T) {
	oldColorLevel := color.ForceSetColorLevel(terminfo.ColorLevelNone)
	defer color.ForceSetColorLevel(oldColorLevel)

	rgbStyle := style.New().SetFg(style.NewRGBColor(color.RGB(1, 2, 3)))
	assert.Equal(t, "x", cachedSprint(rgbStyle, "x"))

	color.ForceSetColorLevel(terminfo.ColorLevelMillions)
	assert.Equal(t, rgbStyle.Sprint("x"), cachedSprint(rgbStyle, "x"))
	assert.NotEqual(t, "x", cachedSprint(rgbStyle, "x"))
}
