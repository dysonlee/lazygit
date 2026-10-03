package graph

import (
	"io"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/gookit/color"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
)

const (
	MergeSymbol  = '◎'
	CommitSymbol = '○'
)

// Glyphs determine how a graph is drawn, mainly the strings a graph cell is
// drawn with. A cell is a node or box-drawing character followed by a
// connector to the next cell, which is Horizontal if the cell connects to the
// right and Blank otherwise; the width of the connectors therefore determines
// the spacing between lanes.
type Glyphs struct {
	Commit     string
	Merge      string
	Horizontal string
	Blank      string
	// A cell where an edge from the row's commit starts in a vertical line
	// that is there anyway, i.e. where both the line above and the line below
	// connect to the commit. Junction is used when the horizontal line comes
	// from the left and ends in the cell, JunctionFromRight when it comes from
	// the right, and JunctionThrough when it passes through.
	Junction          string
	JunctionFromRight string
	JunctionThrough   string
	// Whether the highlighted lines of the selected commit hide the parts of
	// other lines that they run through, rather than only being drawn in the
	// highlight color.
	SelectionHidesOtherLines bool
	// Whether all rows are padded to the width of the widest one, so that
	// whatever follows the graph lines up in a column.
	AlignRows bool
}

var ClassicGlyphs = &Glyphs{
	Commit:            string(CommitSymbol),
	Merge:             string(MergeSymbol),
	Horizontal:        "─",
	Blank:             " ",
	Junction:          "│",
	JunctionFromRight: "│",
	JunctionThrough:   "│",

	SelectionHidesOtherLines: true,
}

// The lane layout uses bigger nodes and wider spacing between lanes, so that
// each branch stands out as a line of its own. Merge commits are small dots
// on the line, so that the commits with actual changes stand out.
//
// A lane can end at a merge commit on the same row where the commit's edge to
// another parent starts in that lane; the junction glyphs show that both of
// them connect to the commit, as opposed to a lane that merely crosses the
// horizontal line.
var LaneGlyphs = &Glyphs{
	Commit:            "●",
	Merge:             "•",
	Horizontal:        "──",
	Blank:             "  ",
	Junction:          "┤",
	JunctionFromRight: "├",
	JunctionThrough:   "┼",

	// Lanes don't move, so the highlight color is enough to tell the selected
	// commit's lines apart, and hiding parts of other lines would make them
	// look as if they ended there.
	SelectionHidesOtherLines: false,

	// The lanes make the graph wide, and the messages are easier to scan
	// when they line up.
	AlignRows: true,
}

// With a Nerd Font, the lane graph uses icon font dots, which fill the cell
// better than the Unicode ones in most fonts. They are at the same code points
// in Nerd Fonts 2 and 3 (where nf-oct-dot_fill used to be called
// nf-oct-primitive_dot).
var LaneNerdFontGlyphs = func() *Glyphs {
	glyphs := *LaneGlyphs
	glyphs.Commit = "\uf111" // nf-fa-circle
	glyphs.Merge = "\uf444"  // nf-oct-dot_fill
	return &glyphs
}()

// LabelLine returns the given number of columns of the line that connects a
// label in front of the graph to its commit. It is dotted, so that it can't be
// mistaken for a branch.
func LabelLine(width int) string {
	return strings.Repeat("┈", width)
}

type cellType int

const (
	CONNECTION cellType = iota
	COMMIT
	MERGE
)

type Cell struct {
	up, down, left, right bool
	junction              bool
	// whether the horizontal line through the cell, or the connector to the
	// next cell, is only part of the line that connects a label to its commit
	labelLine, labelLineRight bool
	cellType                  cellType
	rightStyle                *style.TextStyle
	style                     *style.TextStyle
}

// WithBranchDrawingGlyphs returns the given glyphs with the junctions drawn
// with branch drawing characters (U+F5D0-U+F60D, as defined by flog-symbols),
// which have rounded corners. Only some
// terminals draw these characters themselves, e.g. kitty and Ghostty; in
// others they are private use characters that most fonts don't have.
func WithBranchDrawingGlyphs(glyphs *Glyphs) *Glyphs {
	result := *glyphs
	result.Junction = "\uf5df"          // ╮ and ╯
	result.JunctionFromRight = "\uf5dc" // ╭ and ╰
	result.JunctionThrough = "\uf5e8"   // ─, ╮ and ╯
	return &result
}

func (cell *Cell) render(writer io.StringWriter, glyphs *Glyphs) {
	var first string
	switch cell.cellType {
	case CONNECTION:
		if cell.junction && cell.up && cell.down {
			first = junctionGlyph(cell.left, cell.right, glyphs)
		} else if cell.labelLine {
			first = LabelLine(1)
		} else {
			first = getBoxDrawingChar(cell.up, cell.down, cell.left, cell.right)
		}
	case COMMIT:
		first = glyphs.Commit
	case MERGE:
		first = glyphs.Merge
	}

	var rightStyle *style.TextStyle
	if cell.rightStyle == nil {
		rightStyle = cell.style
	} else {
		rightStyle = cell.rightStyle
	}

	// just doing this for the sake of easy testing, so that we don't need to
	// assert on the style of a space given a space has no styling (assuming we
	// stick to only using foreground styles)
	var styledSecond string
	if cell.right {
		if cell.labelLineRight {
			styledSecond = cachedSprint(*rightStyle, LabelLine(utf8.RuneCountInString(glyphs.Horizontal)))
		} else {
			styledSecond = cachedSprint(*rightStyle, glyphs.Horizontal)
		}
	} else {
		styledSecond = glyphs.Blank
	}

	_, _ = writer.WriteString(cachedSprint(*cell.style, first))
	_, _ = writer.WriteString(styledSecond)
}

// The rendered string depends on the color level too, which tests change.
type rgbCacheKey struct {
	*color.RGBStyle
	str   string
	level color.Level
}

var (
	rgbCache      = make(map[rgbCacheKey]string)
	rgbCacheMutex sync.RWMutex
)

func cachedSprint(style style.TextStyle, str string) string {
	switch v := style.Style.(type) {
	case *color.RGBStyle:
		rgbCacheMutex.RLock()
		key := rgbCacheKey{v, str, color.TermColorLevel()}
		value, ok := rgbCache[key]
		rgbCacheMutex.RUnlock()
		if ok {
			return value
		}
		value = style.Sprint(str)
		rgbCacheMutex.Lock()
		rgbCache[key] = value
		rgbCacheMutex.Unlock()
		return value
	case color.Basic:
		return style.Sprint(str)
	case color.Style:
		value := style.Sprint(str)
		return value
	}
	return style.Sprint(str)
}

func (cell *Cell) reset() {
	cell.up = false
	cell.down = false
	cell.left = false
	cell.right = false
}

func (cell *Cell) setUp(style *style.TextStyle) *Cell {
	cell.up = true
	cell.style = style
	return cell
}

func (cell *Cell) setDown(style *style.TextStyle) *Cell {
	cell.down = true
	cell.style = style
	return cell
}

func (cell *Cell) setLeft(style *style.TextStyle) *Cell {
	cell.left = true
	if !cell.up && !cell.down {
		// vertical trumps left
		cell.style = style
	}
	return cell
}

//nolint:unparam
func (cell *Cell) setRight(style *style.TextStyle, override bool) *Cell {
	cell.right = true
	if cell.rightStyle == nil || override {
		cell.rightStyle = style
	}
	return cell
}

// connectHorizontally adds a line through the cell from left to right, in the
// given style unless the cell already has lines of its own.
func (cell *Cell) connectHorizontally(style *style.TextStyle) {
	if !cell.up && !cell.down && !cell.left && !cell.right {
		cell.style = style
		cell.labelLine = true
	}
	cell.left = true
	if !cell.right {
		cell.right = true
		cell.labelLineRight = true
		cell.rightStyle = style
	}
}

func (cell *Cell) setStyle(style *style.TextStyle) *Cell {
	cell.style = style
	return cell
}

func (cell *Cell) setJunction() *Cell {
	cell.junction = true
	return cell
}

func (cell *Cell) setType(cellType cellType) *Cell {
	cell.cellType = cellType
	return cell
}

func junctionGlyph(left, right bool, glyphs *Glyphs) string {
	if left && right {
		return glyphs.JunctionThrough
	} else if right {
		return glyphs.JunctionFromRight
	}
	return glyphs.Junction
}

func getBoxDrawingChar(up, down, left, right bool) string {
	if up && down {
		return "│"
	} else if up && !down && left && right {
		return "┴"
	} else if up && !down && left && !right {
		return "╯"
	} else if up && !down && !left && right {
		return "╰"
	} else if up && !down && !left && !right {
		return "╵"
	} else if !up && down && left && right {
		return "┬"
	} else if !up && down && left && !right {
		return "╮"
	} else if !up && down && !left && right {
		return "╭"
	} else if !up && down && !left && !right {
		return "╷"
	} else if left {
		return "─"
	} else if right {
		return "╶"
	}

	return " "
}
