package term

import (
	"io"

	"github.com/gdamore/tcell"
	"github.com/mibk/syd/ui"
)

type UI struct {
	screen        tcell.Screen
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

	go t.translateEvents()
	return nil
}

func (t *UI) Close() error {
	t.screen.Fini()
	return nil
}

func (t *UI) Size() (w, h int) { return t.screen.Size() }

func (t *UI) NewWindow() *Window {
	return &Window{
		x: 1, y: 1, // For testing purposes.
		ui:    t,
		frame: new(Frame),
		bgstyle: tcell.StyleDefault.
			Background(tcell.GetColor("#ffffea")),
		hlstyle: tcell.StyleDefault.
			Background(tcell.GetColor("#dfdf9f")),
	}
}

type Window struct {
	ui    *UI
	frame *Frame

	width, height int
	x, y          int

	cur struct {
		p0, p1 int // char position
		x, y   int // current position
	}

	// styles
	bgstyle tcell.Style
	hlstyle tcell.Style
}

func (win *Window) Size() (w, h int) {
	// TODO: Return the width and height of the window.
	w, h = win.ui.Size()
	return w / 2, h
}

func (win *Window) Position() (x, y int) {
	return win.x, win.y
}

func (win *Window) Clear() {
	// TODO: Clean only the window portion.
	win.ui.screen.Clear()
	*win.frame = Frame{
		lines:   make([][]rune, 1),
		wantCol: win.frame.wantCol,
	}
	win.cur.x, win.cur.y = 0, 0
	win.checkSelection()
}

func (win *Window) Select(p0, p1 int) { win.cur.p0, win.cur.p1 = p0, p1 }

func (win *Window) WriteRune(r rune) error {
	if r != '\n' {
		win.frame.lines[win.cur.y] = append(win.frame.lines[win.cur.y], r)
	}

	w, h := win.Size()
	if win.cur.x >= w || r == '\n' {
		win.cur.y++
		win.cur.x = 0
		win.frame.lines = append(win.frame.lines, nil)
		if win.cur.y == h {
			return io.EOF
		}
	} else if r == '\t' {
		win.cur.x += tabWidthForCol(win.cur.x)
	} else {
		win.cur.x++
	}
	win.frame.nchars++
	win.checkSelection()
	return nil
}

// checkSelection tries to line0, line1, and wantCol.
func (win *Window) checkSelection() {
	if win.cur.p0 == win.frame.nchars {
		win.frame.line0 = win.cur.y
		if win.frame.wantCol == ui.ColQ0 {
			win.frame.wantCol = win.cur.x
		}
	}
	if win.cur.p1 == win.frame.nchars {
		win.frame.line1 = win.cur.y
		if win.frame.wantCol == ui.ColQ1 {
			win.frame.wantCol = win.cur.x
		}
	}
}

func (win *Window) Flush() {
	width, _ := win.Size()
	win.ui.screen.Fill(' ', win.bgstyle)
	style := win.bgstyle
	selText := func(p, x, y int) {
		if p == win.cur.p0 {
			if win.cur.p0 == win.cur.p1 {
				win.ui.screen.ShowCursor(win.x+x, win.y+y)
			} else {
				style = win.hlstyle
			}
		} else if p == win.cur.p1 {
			style = win.bgstyle
		}
	}
	win.ui.screen.HideCursor()
	p := 0
	for y, l := range win.frame.lines {
		x := 0
		for _, r := range l {
			selText(p, x, y)
			w := 1
			if r == '\t' {
				r = ' '
				w = tabWidthForCol(x)

			}
			for i := 0; i < w; i++ {
				win.ui.screen.SetContent(win.x+x, win.y+y, r, nil, style)
				x += 1
			}
			p++
		}
		selText(p, x, y)
		for ; x < width; x++ {
			win.ui.screen.SetContent(win.x+x, win.y+y, ' ', nil, style)
		}
		p++
	}
	win.ui.screen.Show()
}

func (win *Window) Frame() ui.Frame { return win.frame }

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
	return 0
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
