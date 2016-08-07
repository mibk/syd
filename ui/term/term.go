package term

import (
	"io"

	"github.com/gdamore/tcell"
	"github.com/mibk/syd/ui"
)

type UI struct {
	screen tcell.Screen
	frame  *Frame

	p0, p1 int // cursor position
	x, y   int // current position

	wasBtnPressed bool
}

func (t *UI) Init() error {
	sc, err := tcell.NewScreen()
	if err != nil {
		return err
	}
	if err := sc.Init(); err != nil {
		return err
	}
	sc.EnableMouse()
	t.screen = sc
	t.frame = new(Frame)
	go t.translateEvents()
	return nil
}

func (t *UI) Close() error {
	t.screen.Fini()
	return nil
}

func (t *UI) Size() (w, h int) { return t.screen.Size() }

func (t *UI) Clear() {
	t.screen.Clear()
	*t.frame = Frame{
		lines:   make([][]rune, 1),
		wantCol: t.frame.wantCol,
	}
	t.x, t.y = 0, 0
	t.checkSelection()
}

func (t *UI) Select(p0, p1 int) { t.p0, t.p1 = p0, p1 }

func (t *UI) WriteRune(r rune) error {
	if r != '\n' {
		t.frame.lines[t.y] = append(t.frame.lines[t.y], r)
	}

	w, h := t.Size()
	if t.x >= w || r == '\n' {
		t.y++
		t.x = 0
		t.frame.lines = append(t.frame.lines, nil)
		if t.y == h {
			return io.EOF
		}
	} else if r == '\t' {
		t.x += tabWidthForCol(t.x)
	} else {
		t.x++
	}
	t.frame.nchars++
	t.checkSelection()
	return nil
}

// checkSelection tries to line0, line1, and wantCol.
func (t *UI) checkSelection() {
	if t.p0 == t.frame.nchars {
		t.frame.line0 = t.y
		if t.frame.wantCol == ui.ColQ0 {
			t.frame.wantCol = t.x
		}
	}
	if t.p1 == t.frame.nchars {
		t.frame.line1 = t.y
		if t.frame.wantCol == ui.ColQ1 {
			t.frame.wantCol = t.x
		}
	}
}

func (t *UI) Flush() {
	st := tcell.StyleDefault
	selText := func(p, x, y int) {
		if p == t.p0 {
			if t.p0 == t.p1 {
				t.screen.ShowCursor(x, y)
			} else {
				st = st.Reverse(true)
			}
		} else if p == t.p1 {
			st = st.Reverse(false)
		}
	}
	t.screen.HideCursor()
	p := 0
	for y, l := range t.frame.lines {
		x := 0
		for _, r := range l {
			selText(p, x, y)
			t.screen.SetContent(x, y, r, nil, st)
			p++
			if r == '\t' {
				x += tabWidthForCol(x)
			} else {
				x++
			}
		}
		selText(p, x, y)
		p++
	}
	t.screen.Show()
}

func (t *UI) Frame() ui.Frame { return t.frame }

type Frame struct {
	lines   [][]rune
	line0   int
	line1   int
	wantCol int
	nchars  int
}

func (f *Frame) Nchars() int                { return f.nchars }
func (f *Frame) SelectionLines() (int, int) { return f.line0, f.line1 }

func (f *Frame) CharsUntilXY(x, y int) int {
	if y >= len(f.lines) {
		return f.nchars
	}
	var p int
	for n, l := range f.lines {
		if n == y {
			return p + charsUntilX(l, x)
		}
		p += len(l) + 1 // + '\n'
	}
	panic("shouldn't happen")
}

func charsUntilX(s []rune, x int) int {
	var w int
	for i, r := range s {
		if r == '\t' {
			w += tabWidthForCol(w)
		} else {
			w += 1
		}
		if w > x {
			return i
		}
	}
	return len(s)
}

const tabStop = 8

func tabWidthForCol(col int) int {
	w := tabStop - col%tabStop
	if w == 0 {
		return tabStop
	}
	return w
}

func (f *Frame) MaxLines() int { panic("not implemented") }
func (f *Frame) Lines() int    { return len(f.lines) }

func (f *Frame) WantCol() int       { return f.wantCol }
func (f *Frame) SetWantCol(col int) { f.wantCol = col }
