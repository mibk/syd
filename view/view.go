package view

import (
	"io"
	"unicode/utf8"

	"github.com/mibk/syd/ui/console"
)

const tabStop = 8

type View struct {
	reader io.ReaderAt
	offset int64
	lines  []*Line

	// current line number relative to the offset
	line int
	// current cell
	cell   int
	maxCol int
}

func New(r io.ReaderAt) *View {
	return &View{reader: r}
}

// TODO: rm
func (v *View) ToTheStartColumn() {
	v.cell = 0
}

func (v *View) MoveDown() {
	v.line += 1
	if v.line == len(v.lines) {
		v.line = len(v.lines) - 1
	}
	v.findColumn()
}

func (v *View) MoveUp() {
	v.line -= 1
	if v.line == -1 {
		v.line = 0
	}
	v.findColumn()
}

func (v *View) MoveLeft() {
	if v.cell != 0 {
		v.cell -= 1
		v.maxCol = v.CurrentCell().col
	}
}

func (v *View) MoveRight() {
	if v.cell < len(v.lines[v.line].cells)-1 {
		v.cell += 1
		v.maxCol = v.CurrentCell().col
	}
}

func (v *View) findColumn() {
	cells := v.lines[v.line].cells
	for i, c := range cells {
		if c.col >= v.maxCol {
			v.cell = i
			if c.col > v.maxCol {
				v.cell--
			}
			return
		}
	}
	v.cell = len(cells) - 1
}

func (v *View) ReadLines() {
	buf := make([]byte, 500)
	r := ReaderFrom(v.reader, v.offset)

	start := 0
	v.lines = make([]*Line, 0)
	curLine := new(Line)
	col := 0
	offset := 0

	for {
		pos := 0

		// buf[:start] contains a part of the last rune from a previous
		// decoding. So let it there to optain the whole rune. The value of
		// n is therefore bigger by the value of start.
		n, err := r.Read(buf[start:])
		n += start
		start = 0

		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
		for {
			r, size := utf8.DecodeRune(buf[pos:n])
			if r == utf8.RuneError {
				// TODO: r doesn't have to be the last rune
				copy(buf, buf[pos:n])
				start = n - pos
				break
			} else if r == '\n' {
				curLine.cells = append(curLine.cells, Cell{'\n', offset + pos, col})
				v.lines = append(v.lines, curLine)
				curLine = new(Line)
				col = 0
			} else {
				curLine.cells = append(curLine.cells, Cell{r, offset + pos, col})
				if r == '\t' {
					w := tabStop - col%tabStop
					if w == 0 {
						w = tabStop
					}
					col += w
				} else {
					col++
				}
			}
			pos += size
		}
		offset += pos
	}
	if len(curLine.cells) > 0 {
		v.lines = append(v.lines, curLine)
	}
}

func (v *View) Draw(ui console.Console) {
	v.ReadLines()
	ui.Clear()
	if v.line >= len(v.lines) {
		v.line = len(v.lines) - 1
	}
	col := 0
	cells := v.lines[v.line].cells
	if len(cells) > 0 {
		if v.cell >= len(cells) {
			v.cell = len(cells) - 1
		}
		col = cells[v.cell].col
	}
	ui.SetCursor(col, v.line)

	for y, l := range v.lines {
		for _, cell := range l.cells {
			ui.SetCell(cell.col, y, cell.R, console.AttrDefault)
		}
	}
}

func (v *View) CurrentCell() Cell {
	return v.lines[v.line].cells[v.cell]
}

type Line struct {
	cells []Cell
}

type Cell struct {
	R   rune
	Off int
	col int
}
