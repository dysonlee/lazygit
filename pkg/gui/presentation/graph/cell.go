package graph

import (
	"io"
	"sync"

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
// each branch stands out as a line of its own.
//
// A lane can end at a merge commit on the same row where the commit's edge to
// another parent starts in that lane; the junction glyphs show that both of
// them connect to the commit, as opposed to a lane that merely crosses the
// horizontal line.
var LaneGlyphs = &Glyphs{
	Commit:            "●",
	Merge:             "◉",
	Horizontal:        "──",
	Blank:             "  ",
	Junction:          "┤",
	JunctionFromRight: "├",
	JunctionThrough:   "┼",

	// Lanes don't move, so the highlight color is enough to tell the selected
	// commit's lines apart, and hiding parts of other lines would make them
	// look as if they ended there.
	SelectionHidesOtherLines: false,
}

// With a Nerd Font, the lane graph uses Font Awesome's circles, which are
// bigger than the Unicode ones in most fonts. They are at the same code
// points in Nerd Fonts 2 and 3.
var LaneNerdFontGlyphs = func() *Glyphs {
	glyphs := *LaneGlyphs
	glyphs.Commit = "\uf111" // nf-fa-circle
	glyphs.Merge = "\uf192"  // nf-fa-dot_circle_o
	return &glyphs
}()

type cellType int

const (
	CONNECTION cellType = iota
	COMMIT
	MERGE
)

type Cell struct {
	up, down, left, right bool
	junction              bool
	cellType              cellType
	rightStyle            *style.TextStyle
	style                 *style.TextStyle
}

func (cell *Cell) render(writer io.StringWriter, glyphs *Glyphs) {
	var first string
	switch cell.cellType {
	case CONNECTION:
		if cell.junction && cell.up && cell.down {
			first = junctionGlyph(cell.left, cell.right, glyphs)
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
		styledSecond = cachedSprint(*rightStyle, glyphs.Horizontal)
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
