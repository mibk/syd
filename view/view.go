package view

import (
	"io"
	"unicode/utf8"

	"github.com/mibk/syd/ui/console"
)

// Last is used to denote for example last line
// or last column.
const Last = -2

const tabStop = 8

type View struct {
	height int
	reader io.ReaderAt
	offset int64
	lines  []*Line

	firstLine int
	line      int
	// current cell
	cell   int
	maxCol int
}

func New(r io.ReaderAt) *View {
	return &View{reader: r}
}

func (v *View) SetHeight(h int) {
	v.height = h
}

func (v *View) GotoLine(n int) {
	if n == Last {
		n = len(v.lines) - 1
	} else if n < 0 {
		n = 0
	} else if n > len(v.lines)-1 {
		n = len(v.lines) - 1
	}
	v.line = n
	l := v.screenLine()
	if l < 0 {
		v.firstLine += l
	} else if l > v.height-1 {
		v.firstLine += l - (v.height - 1)
	}
	v.findColumn()
}

func (v *View) findColumn() {
	cells := v.lines[v.line].cells
	for i, c := range cells {
		if c.column >= v.maxCol {
			v.cell = i
			if c.column > v.maxCol {
				v.cell--
			}
			return
		}
	}
	v.cell = len(cells) - 1
}

func (v *View) Line() int {
	return v.line
}

func (v *View) screenLine() int {
	return v.line - v.firstLine
}

func (v *View) GotoColumn(n int) {
	if n == Last {
		n = len(v.lines[v.line].cells) - 1
	} else if n < 0 {
		n = 0
	} else if n > len(v.lines[v.line].cells)-1 {
		n = len(v.lines[v.line].cells) - 1
	}
	v.cell = n
	v.maxCol = v.CurrentCell().column
}

func (v *View) Column() int {
	return v.cell
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

	col := 0
	cells := v.lines[v.line].cells
	if len(cells) > 0 {
		if v.cell >= len(cells) {
			v.cell = len(cells) - 1
		}
		col = cells[v.cell].column
	}
	ui.SetCursor(col, v.screenLine())

	for y := 0; y < v.height; y++ {
		if y+v.firstLine > len(v.lines)-1 {
			break
		}
		l := v.lines[y+v.firstLine]
		for _, cell := range l.cells {
			ui.SetCell(cell.column, y, cell.Rune, console.AttrDefault)
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
	Rune   rune
	Offset int
	column int
}
