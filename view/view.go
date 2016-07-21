package view

import (
	"io"

	"github.com/mibk/syd/core"
	"github.com/mibk/syd/event"
	"github.com/mibk/syd/ui/console"
)

const tabStop = 8

type View struct {
	width  int
	height int
	buf    *core.Buffer

	origin int64
	q0, q1 int64

	// Frame
	lines       [][]rune
	line0, col0 int
	line1, col1 int
	wantCol     int
	nchars      int
}

func New(buf *core.Buffer) *View {
	return &View{buf: buf}
}

// Size returns the size of v.
func (v *View) Size() (int, int) {
	return v.height, v.width
}

// SetSize sets the size of v.
func (v *View) SetSize(w, h int) {
	v.width, v.height = w, h
}

func (v *View) Render(ui console.Console) {
	setCursor := func(x, y int) {
		ui.SetCursor(x, y)
		v.line0, v.line1 = y, y
		v.col0, v.col1 = x, x
		if v.wantCol == -1 {
			v.wantCol = x
		}
	}
	v.loadText()
	ui.SetCursor(-1, -1)
	ui.Clear()
	p := v.origin
	for y, l := range v.lines {
		x := 0
		for _, r := range l {
			ui.SetCell(x, y, r, console.AttrDefault)
			if p == v.q0 {
				setCursor(x, y)
			}
			p++
			if r == '\t' {
				x += tabWidthForCol(x)
			} else {
				x++
			}
		}
		if p == v.q0 {
			setCursor(x, y)
		}
		p++
	}
}

func (v *View) loadText() {
	v.lines = nil
	x, y := 0, 0
	p := v.origin
	for ; ; p++ {
		if len(v.lines) <= y {
			v.lines = append(v.lines, nil)
		}
		r, _, err := v.buf.ReadRuneAt(p)
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
		if r != '\n' {
			v.lines[y] = append(v.lines[y], r)
		}

		if x >= v.width || r == '\n' {
			y++
			x = 0
			if y == v.height {
				break
			}
		} else if r == '\t' {
			x += tabWidthForCol(x)
		} else {
			x++
		}
	}
	v.nchars = int(p - v.origin)
}

func tabWidthForCol(col int) int {
	w := tabStop - col%tabStop
	if w == 0 {
		return tabStop
	}
	return w
}

func (v *View) Type(ev event.KeyPress) {
	switch {
	case ev.Key == event.Escape:
	case ev.Key == event.Backspace:
		if v.q0 == 0 {
			return
		}
		v.q0, v.q1 = v.q0-1, v.q0-1
		fallthrough
	case ev.Key == event.Delete:
		v.buf.Delete(v.q0, v.q0+1)
		v.checkVisibility()
	case ev.Key == event.Left:
		if v.q0 == 0 {
			return
		}
		v.q0, v.q1 = v.q0-1, v.q0-1
		v.wantCol = -1
		v.checkVisibility()
	case ev.Key == event.Right:
		v.q0, v.q1 = v.q0+1, v.q0+1
		v.wantCol = -1
		if v.q0 > v.origin+int64(v.nchars) {
			oldOrg := v.origin
			v.origin = v.nextNewLine(3)
			v.loadText()
			if v.q0 > v.origin+int64(v.nchars) {
				// There's no more content, get back.
				v.origin = oldOrg
				v.q0, v.q1 = v.q0-1, v.q0-1
				v.loadText()
			}
		}
		v.checkVisibility()
	case ev.Key == event.Up:
		q := v.findQ(v.line0-1, v.wantCol)
		v.q0, v.q1 = q, q
	case ev.Key == event.Down:
		q := v.findQ(v.line1+1, v.wantCol)
		v.q0, v.q1 = q, q

	// Temporary shortcuts:
	case ev.Key == 'z' && ev.Ctrl:
		v.buf.Undo()
	case ev.Key == 'y' && ev.Ctrl:
		v.buf.Redo()
	case ev.Key == event.PageUp:
		v.origin = v.prevNewLine(v.origin, v.height)
	case ev.Key == event.PageDown:
		v.origin = v.origin + int64(v.nchars)

	default:
		v.buf.Insert(v.q0, string(ev.Key))
		v.q0, v.q1 = v.q0+1, v.q0+1
		v.wantCol = -1
		v.checkVisibility()
	}
}

func (v *View) checkVisibility() {
	if v.q0 < v.origin || v.q0 > v.origin+int64(v.nchars)+1 {
		v.origin = v.prevNewLine(v.q0, 3)
	}
}

func (v *View) prevNewLine(p int64, n int) int64 {
	for ; n > 0; n-- {
		// Shorten long lines. After 128 characters call it a line anyway.
		for i := 0; i < 128 && p > 0; i++ {
			p--
			if p == 0 {
				return 0
			}
			r, _, err := v.buf.ReadRuneAt(p - 1)
			if err != nil {
				panic(err)
			}
			if r == '\n' {
				break
			}
		}
	}
	return p
}

func (v *View) nextNewLine(n int) int64 {
	c := 0
	for _, l := range v.lines {
		c += len(l) + 1 // + '\n'
		n--
		if n == 0 {
			goto NotLastLine
		}
	}
	c-- // last line doesn't contain '\n'
NotLastLine:
	return v.origin + int64(c)
}

func (v *View) findQ(line, col int) int64 {
	if line < 0 {
		v.origin = v.prevNewLine(v.origin, -line)
		v.loadText()
		line = 0
	} else if line > len(v.lines)-1 {
		if len(v.lines) == v.height {
			i := line - len(v.lines) + 1
			oldOrg := v.origin
			l := len(v.lines)
			v.origin = v.nextNewLine(i)
			v.loadText()
			if len(v.lines) < l {
				v.origin = oldOrg
				v.loadText()
			}
		}
		line = len(v.lines) - 1
	}
	q := v.origin
	for n, l := range v.lines {
		if n < line {
			q += int64(len(l)) + 1 // + '\n'
			continue
		}
		x := 0
		for i, r := range v.lines[n] {
			if r == '\t' {
				x += tabWidthForCol(x)
			} else {
				x += 1
			}
			if x > col {
				return q + int64(i)
			}
		}
		return q + int64(len(v.lines[n]))
	}
	panic("shouldn't happen")
}
