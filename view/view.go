package view

import (
	"io"
	"unicode/utf8"

	"github.com/mibk/syd/ui/console"
)

// Last is used to denote for example last line or last column.
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
	cell       int
	desiredCol int
}

func New(r io.ReaderAt) *View {
	return &View{reader: r}
}

func (v *View) Height() int {
	return v.height
}

func (v *View) SetHeight(h int) {
	v.height = h
}

func (v *View) GotoLine(n int) {
	if n == Last {
		n = len(v.lines) - 1
	} else {
		n = v.validateLineNumber(n)
	}
	v.line = n
	l := v.ScreenLine()
	if l < 0 {
		v.firstLine += l
	} else if l > v.height-1 {
		v.firstLine += l - (v.height - 1)
	}
	v.findColumn()
}

func (v *View) validateLineNumber(n int) int {
	if n < 0 {
		return 0
	} else if n > len(v.lines)-1 {
		return len(v.lines) - 1
	}
	return n
}

func (v *View) findColumn() {
	cells := v.lines[v.line].cells
	for i, c := range cells {
		if c.column >= v.desiredCol {
			v.cell = i
			if c.column > v.desiredCol {
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

func (v *View) ScreenLine() int {
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
	v.desiredCol = v.CurrentCell().column
}

func (v *View) Column() int {
	return v.cell
}

func (v *View) FirstLine() int {
	return v.firstLine
}

func (v *View) SetFirstLine(n int) {
	n = v.validateLineNumber(n)
	l := v.ScreenLine()
	v.firstLine = n
	v.line = n + l
	v.line = v.validateLineNumber(v.line)
	v.findColumn()
}

// SetCursor sets the cursor to the specified offset in the buffer.
func (v *View) SetCursor(offset int) {
	for line, l := range v.lines {
		for col, c := range l.cells {
			if c.Offset >= offset {
				v.GotoLine(line)
				v.GotoColumn(col)
				return
			}
		}
	}
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

	if v.line > len(v.lines)-1 {
		v.line = len(v.lines) - 1
		v.cell = len(v.lines[v.line].cells) - 1
		v.desiredCol = v.lines[v.line].cells[v.cell].column
	}

	col := 0
	cells := v.lines[v.line].cells
	if len(cells) > 0 {
		if v.cell >= len(cells) {
			v.cell = len(cells) - 1
		}
		col = cells[v.cell].column
	}
	ui.SetCursor(col, v.ScreenLine())

	y := 0
	for ; y < v.height; y++ {
		if y+v.firstLine > len(v.lines)-1 {
			break
		}
		l := v.lines[y+v.firstLine]
		for _, cell := range l.cells {
			ui.SetCell(cell.column, y, cell.Rune, console.AttrDefault)
		}
	}
	for ; y < v.height; y++ {
		ui.SetCell(0, y, '~', console.AttrDefault)
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
