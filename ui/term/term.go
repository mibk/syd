package term

import (
	"io"

	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/mouse"

	"github.com/gdamore/tcell"
	"github.com/mibk/syd/ui"
)

type UI struct {
	screen        tcell.Screen
	wasBtnPressed bool

	windows    []*Window
	activeText *Text // will receive key events
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
			Background(tcell.GetColor("#90e0e0")),
	}
	body := &Text{
		frame: new(Frame),
		bgstyle: tcell.StyleDefault.
			Background(tcell.GetColor("#ffffea")),
		hlstyle: tcell.StyleDefault.
			Background(tcell.GetColor("#e0e090")),
	}
	w, h := t.Size()
	win := &Window{
		x: 1, y: 1, // For testing purposes.
		width: w / 2,
		ui:    t,
		head:  head,
		body:  body,
	}
	head.win = win
	body.win = win

	if cnt := len(t.windows); cnt == 0 {
		t.activeText = body
		win.height = h - 2 // TODO: Just for testing.
	} else {
		prev := t.windows[cnt-1]
		win.height = prev.height / 2
		prev.height -= prev.height / 2
		win.y = prev.y + prev.height
	}
	t.windows = append(t.windows, win)
	return win
}

// TODO: This is for temporary reasons. Remove it.
func (t *UI) Push_Mouse_Event(ev mouse.Event) {
	y := int(ev.Y)
	for _, win := range t.windows {
		if y < win.y || y >= win.y+win.height {
			continue
		}
		if y >= win.body.y {
			win.body.click(ev)
			t.activeText = win.body
		} else {
			win.head.click(ev)
			t.activeText = win.head
		}
		break
	}
}

func (t *UI) Push_Key_Event(ev key.Event) {
	t.activeText.keyEventHandler(ev)
}

type Window struct {
	ui *UI

	width, height int
	x, y          int

	head *Text
	body *Text
}

func (win *Window) Size() (w, h int) {
	return win.width, win.height
}

func (win *Window) Head() *Text { return win.head }
func (win *Window) Body() *Text { return win.body }

func (win *Window) Clear() {
	win.head.width = win.width
	win.head.height = win.height
	win.head.clear()

	win.body.width = win.width
	win.body.height = win.height
	win.body.clear()
}

func (win *Window) Flush() {
	win.head.x = win.x
	win.head.y = win.y
	win.head.flush()

	h := len(win.head.frame.lines)
	win.head.height = h

	win.body.height = win.height - h
	if len(win.body.frame.lines) > win.body.height {
		// TODO: We didn't know how many lines will the head of the window
		// span. Can we do better?
		win.body.frame.lines = win.body.frame.lines[:win.body.height]
	}
	win.body.x = win.x
	win.body.y = win.y + h
	win.body.flush()
	win.body.fill()

	win.ui.screen.Show()
}

type Text struct {
	win   *Window
	frame *Frame

	width, height int
	x, y          int
	cur           struct {
		p0, p1 int // char position
		x, y   int // current position
	}

	// styles
	bgstyle tcell.Style
	hlstyle tcell.Style

	mouseEventHandler ui.MouseEventHandler
	keyEventHandler   ui.KeyEventHandler
}

func (t *Text) click(ev mouse.Event) {
	if t.mouseEventHandler == nil {
		return
	}
	p := t.frame.CharsUntilXY(int(ev.X)-t.x, int(ev.Y)-t.y)
	t.mouseEventHandler(p, ev)
}

func (t *Text) OnMouseEvent(h ui.MouseEventHandler) {
	t.mouseEventHandler = h
}

func (t *Text) OnKeyEvent(h ui.KeyEventHandler) {
	t.keyEventHandler = h
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
	t.frame.lines[t.cur.y] = append(t.frame.lines[t.cur.y], r)
	if r == '\t' {
		t.cur.x += tabWidthForCol(t.cur.x)
	} else {
		t.cur.x++
	}

	if t.cur.x >= t.width || r == '\n' {
		t.cur.y++
		t.cur.x = 0
		t.frame.lines = append(t.frame.lines, nil)
		if t.cur.y == t.height {
			return io.EOF
		}
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

var reverse = tcell.StyleDefault.Reverse(true)

func (t *Text) flush() {
	style := t.bgstyle
	selStyle := func(p int) {
		if p == t.cur.p0 && t.cur.p0 == t.cur.p1 {
			style = reverse
		} else if p >= t.cur.p0 && p < t.cur.p1 {
			style = t.hlstyle
		} else {
			style = t.bgstyle
		}
	}
	p := 0
	for y, l := range t.frame.lines {
		x := 0
		for _, r := range l {
			selStyle(p)
			p++
			if r == '\n' {
				goto fill
			}
			w := 1
			if r == '\t' {
				r = ' '
				w = tabWidthForCol(x)
			}
			for i := 0; i < w && x < t.width; i++ {
				// TODO: Should the rest of the tab at the end of a
				// line span the begining of the next line?
				t.win.ui.screen.SetContent(t.x+x, t.y+y, r, nil, style)
				x++
				if style == reverse {
					style = t.bgstyle
				}
			}
		}
		selStyle(p)
	fill:
		for ; x < t.width; x++ {
			t.win.ui.screen.SetContent(t.x+x, t.y+y, ' ', nil, style)
			if style == reverse {
				style = t.bgstyle
			}
		}
	}
}

func (t *Text) fill() {
	// TODO: Using this bg color just for testing purposes.
	bg := tcell.StyleDefault.Background(tcell.GetColor("#ffe0ff"))
	for y := len(t.frame.lines); y < t.height; y++ {
		for x := 0; x < t.width; x++ {
			t.win.ui.screen.SetContent(t.x+x, t.y+y, ' ', nil, bg)
		}
	}
}

func (t *Text) Frame() *Frame { return t.frame }

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
		p += len(l)
	}
	return 0
}

func charsUntilX(s []rune, x int) int {
	if len(s) == 0 {
		return 0
	}
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
	if s[len(s)-1] == '\n' {
		return len(s) - 1
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

func (f *Frame) Lines() int         { return len(f.lines) }
func (f *Frame) WantCol() int       { return f.wantCol }
func (f *Frame) SetWantCol(col int) { f.wantCol = col }
