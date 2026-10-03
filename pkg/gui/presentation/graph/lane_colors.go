package graph

import (
	"math"

	"github.com/gookit/color"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/samber/lo"
)

// The colors that the lane graph hands out to branches, in turn.
var laneColors = lo.Map(
	[]string{"#15A0BF", "#0669F7", "#8E00C2", "#C517B6", "#D90171", "#CD0101", "#F25D2E", "#F2CA33", "#7BD938", "#2ECE9D"},
	func(hex string, _ int) color.RGBColor { return color.HEX(hex) },
)

// The styles are created once because cachedSprint caches rendered strings
// by style pointer.
var laneStyles = lo.Map(laneColors, func(laneColor color.RGBColor, _ int) *style.TextStyle {
	value := style.New().SetFg(style.NewRGBColor(laneColor))
	return &value
})

var dimmedLaneStyles = lo.Map(laneStyles, func(laneStyle *style.TextStyle, _ int) *style.TextStyle {
	value := laneStyle.SetDim()
	return &value
})

// Labels in front of the graph have the color of their commit's lane as the
// background, with black or white text, whichever is easier to read on it.
var laneLabelStyles = lo.Map(laneColors, func(laneColor color.RGBColor, _ int) *style.TextStyle {
	textColor := lo.Ternary(isLightColor(laneColor), color.RGB(0, 0, 0), color.RGB(255, 255, 255))
	value := style.New().SetFg(style.NewRGBColor(textColor)).SetBg(style.NewRGBColor(laneColor))
	return &value
})

// The color index of each lane style, dimmed or not
var laneStyleColors = func() map[*style.TextStyle]int {
	result := make(map[*style.TextStyle]int, 2*len(laneStyles))
	for i := range laneStyles {
		result[laneStyles[i]] = i
		result[dimmedLaneStyles[i]] = i
	}
	return result
}()

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

// LaneLabelStyle returns the style for a label of a commit in the lane graph,
// given the style of the commit (see CommitStyle), or nil if it isn't one of
// the lane colors.
func LaneLabelStyle(commitStyle *style.TextStyle) *style.TextStyle {
	laneColor, isLaneStyle := laneStyleColors[commitStyle]
	if !isLaneStyle {
		return nil
	}

	return laneLabelStyles[laneColor]
}

// isLightColor tells whether black text is easier to read on the color than
// white text, using the relative luminance of the color as defined by WCAG.
func isLightColor(rgb color.RGBColor) bool {
	linear := func(channel uint8) float64 {
		value := float64(channel) / 255
		if value <= 0.03928 {
			return value / 12.92
		}
		return math.Pow((value+0.055)/1.055, 2.4)
	}
	values := rgb.Values()
	luminance := 0.2126*linear(uint8(values[0])) + 0.7152*linear(uint8(values[1])) + 0.0722*linear(uint8(values[2]))
	// the luminance at which black and white text have the same contrast ratio
	return luminance > 0.179
}
