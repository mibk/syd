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
	head := &Text{
		frame: new(Frame),
		bgstyle: tcell.StyleDefault.
			Background(tcell.GetColor("#eaffff")),
		hlstyle: tcell.StyleDefault.
			Background(tcell.GetColor("#dfdf9f")),
	}
	body := &Text{
		frame: new(Frame),
		bgstyle: tcell.StyleDefault.
			Background(tcell.GetColor("#ffffea")),
		hlstyle: tcell.StyleDefault.
			Background(tcell.GetColor("#dfdf9f")),
	}
	win := &Window{
		x: 1, y: 1, // For testing purposes.
		ui:   t,
		head: head,
		body: body,
	}
	head.win = win
	body.win = win
	return win
}

type Window struct {
	ui *UI

	width, height int
	x, y          int

	head *Text
	body *Text
}

func (win *Window) Size() (w, h int) {
	// TODO: Return the width and height of the window.
	w, h = win.ui.Size()
	return w / 2, h
}

func (win *Window) Position() (x, y int) {
	return win.x, win.y
}

func (win *Window) Head() ui.Text { return win.head }
func (win *Window) Body() ui.Text { return win.body }

func (win *Window) Clear() {
	win.head.clear()
	win.body.clear()
}

func (win *Window) Flush() {
	_, height := win.Size()
	win.head.x = win.x
	win.head.y = win.y
	win.body.x = win.x
	win.head.flush()
	win.head.fill(ui.HeadHeight + 1)
	win.body.y = win.y + ui.HeadHeight
	win.ui.screen.HideCursor()
	win.body.flush()
	win.body.fill(height)
	win.ui.screen.Show()
}

type Text struct {
	win   *Window
	frame *Frame

	x, y int
	cur  struct {
		p0, p1 int // char position
		x, y   int // current position
	}

	// styles
	bgstyle tcell.Style
	hlstyle tcell.Style
}

func (t *Text) clear() {
	*t.frame = Frame{
		lines:   make([][]rune, 1),
		wantCol: t.frame.wantCol,
	}
	t.cur.x, t.cur.y = 0, 0
	t.checkSelection()
}

func (t *Text) Select(p0, p1 int) { t.cur.p0, t.cur.p1 = p0, p1 }

func (t *Text) WriteRune(r rune) error {
	if r != '\n' {
		t.frame.lines[t.cur.y] = append(t.frame.lines[t.cur.y], r)
	}

	w, h := t.win.Size()
	if t.cur.x >= w || r == '\n' {
		t.cur.y++
		t.cur.x = 0
		t.frame.lines = append(t.frame.lines, nil)
		if t.cur.y == h {
			return io.EOF
		}
	} else if r == '\t' {
		t.cur.x += tabWidthForCol(t.cur.x)
	} else {
		t.cur.x++
	}
	t.frame.nchars++
	t.checkSelection()
	return nil
}

// checkSelection tries to line0, line1, and wantCol.
func (t *Text) checkSelection() {
	if t.cur.p0 == t.frame.nchars {
		t.frame.line0 = t.cur.y
		if t.frame.wantCol == ui.ColQ0 {
			t.frame.wantCol = t.cur.x
		}
	}
	if t.cur.p1 == t.frame.nchars {
		t.frame.line1 = t.cur.y
		if t.frame.wantCol == ui.ColQ1 {
			t.frame.wantCol = t.cur.x
		}
	}
}

func (t *Text) flush() {
	width, _ := t.win.Size()
	style := t.bgstyle
	selText := func(p, x, y int) {
		if p == t.cur.p0 {
			if t.cur.p0 == t.cur.p1 {
				t.win.ui.screen.ShowCursor(t.x+x, t.y+y)
			} else {
				style = t.hlstyle
			}
		} else if p == t.cur.p1 {
			style = t.bgstyle
		}
	}
	p := 0
	for y, l := range t.frame.lines {
		x := 0
		for _, r := range l {
			selText(p, x, y)
			w := 1
			if r == '\t' {
				r = ' '
				w = tabWidthForCol(x)

			}
			for i := 0; i < w; i++ {
				t.win.ui.screen.SetContent(t.x+x, t.y+y, r, nil, style)
				x += 1
			}
			p++
		}
		selText(p, x, y)
		for ; x < width; x++ {
			t.win.ui.screen.SetContent(t.x+x, t.y+y, ' ', nil, style)
		}
		p++
	}
}

func (t *Text) fill(height int) {
	width, _ := t.win.Size()
	for y := len(t.frame.lines) + t.y; y < height; y++ {
		for x := 0; x < width; x++ {
			t.win.ui.screen.SetContent(t.win.x+x, y, ' ', nil, t.bgstyle)
		}
	}
}

func (t *Text) Frame() ui.Frame { return t.frame }

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
