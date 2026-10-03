package graph

import (
	"github.com/gookit/color"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/samber/lo"
)

// The colors that the lane graph hands out to branches, in turn.
//
// The styles are created once because cachedSprint caches rendered strings
// by style pointer.
var laneStyles = lo.Map(
	[]string{"#15A0BF", "#0669F7", "#8E00C2", "#C517B6", "#D90171", "#CD0101", "#F25D2E", "#F2CA33", "#7BD938", "#2ECE9D"},
	func(hex string, _ int) *style.TextStyle {
		value := style.New().SetFg(style.NewRGBColor(color.HEX(hex)))
		return &value
	},
)

var dimmedLaneStyles = lo.Map(laneStyles, func(laneStyle *style.TextStyle, _ int) *style.TextStyle {
	value := laneStyle.SetDim()
	return &value
})

var LaneColorCount = len(laneStyles)

// LaneStyle returns the style of the lane graph's color with the given index,
// which is in the range [0, LaneColorCount).
func LaneStyle(index int) *style.TextStyle {
	return laneStyles[index]
}

// DimmedLaneStyle is like LaneStyle, but for parts of the graph that are drawn
// in faint text.
func DimmedLaneStyle(index int) *style.TextStyle {
	return dimmedLaneStyles[index]
}

// pickLaneColor returns the first color starting from next that isn't in
// use, or next itself if all of them are, along with the color to start from
// the next time.
func pickLaneColor(next int, inUse func(color int) bool) (int, int) {
	for offset := range LaneColorCount {
		candidate := (next + offset) % LaneColorCount
		if !inUse(candidate) {
			return candidate, (candidate + 1) % LaneColorCount
		}
	}

	return next, (next + 1) % LaneColorCount
}
