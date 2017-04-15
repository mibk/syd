package term

import (
	"io"
	"unicode"
	"unicode/utf8"

	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/mouse"

	"github.com/gdamore/tcell"
	"github.com/mibk/syd/ui"
)

var (
	whitebg = tcell.StyleDefault.Background(tcell.ColorWhite)

	dirtystyle  = tcell.StyleDefault.Background(tcell.GetColor("#e5083c"))
	borderstyle = tcell.StyleDefault.Background(tcell.GetColor("#83835c"))

	tagbg  = tcell.StyleDefault.Background(tcell.GetColor("#eaffff"))
	taghl  = tcell.StyleDefault.Background(tcell.GetColor("#90e0e0"))
	bodybg = tcell.StyleDefault.Background(tcell.GetColor("#ffffea"))
	bodyhl = tcell.StyleDefault.Background(tcell.GetColor("#e0e090"))

	testbg = tcell.StyleDefault.Background(tcell.GetColor("#ffe0ff"))
)

type UI struct {
	screen        tcell.Screen
	wasBtnPressed bool

	y      int
	width  int
	height int

	firstCol  *Column
	recentCol *Column // create new windows here

	grabbedCol *Column // grabbed col or nil
	grabbedWin *Window // grabbed win or nil
	activeText *Text   // will receive key events
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

	// TODO: Just for testing purposes.
	t.y = 1
	w, h := t.Size()
	t.width = w - 1
	t.height = h - 2

	go t.translateEvents()
	return nil
}

func (t *UI) Close() error {
	t.screen.Fini()
	return nil
}

func (t *UI) Size() (w, h int) { return t.screen.Size() }

func (t *UI) Flush() {
	col := t.firstCol
	for col != nil {
		col.flush()
		col = col.nextCol
	}
	t.screen.Show()
}

// TODO: This is for temporary reasons. Remove it.
func (t *UI) Push_Mouse_Event(ev mouse.Event) {
	col := t.firstCol
	for col != nil {
		if int(ev.X) < col.x+col.width() {
			col.handleMouseEvent(ev)
			return
		}
		col = col.nextCol
	}
	panic("column not found")
}

func (t *UI) Push_Key_Event(ev key.Event) {
	t.activeText.keyEventHandler(ev)
}

func (t *UI) NewColumn() *Column {
	col := &Column{ui: t}
	if t.firstCol == nil {
		t.firstCol = col
	} else {
		prev := t.lastCol()
		col.x = prev.x + prev.width()/2
		prev.nextCol = col
	}
	return col
}

func (t *UI) NewWindow() *Window {
	if t.recentCol == nil {
		t.recentCol = t.NewColumn()
	}
	return t.recentCol.newWindow()
}

func (t *UI) lastCol() *Column {
	col := t.firstCol
	if col == nil {
		return nil
	}
	for col.nextCol != nil {
		col = col.nextCol
	}
	return col
}

func (t *UI) moveGrabbedCol(x int) {
	gc := t.grabbedCol
	t.grabbedCol = nil

	// TODO: If there are no columns.

	target := t.firstCol
	for target != nil {
		if x >= target.x && x < target.x+target.width() {
			break
		}
		target = target.nextCol
	}

	if x == target.x {
		// TODO: Adjust position. See moveGrabbedWin.
		return
	}

	if gc == target || (target.nextCol != nil && gc == target.nextCol) {
		if gc == t.firstCol {
			return
		}
	} else {
		t.removeCol(gc)
		gc.nextCol = target.nextCol
		target.nextCol = gc
	}
	gc.x = x
}

func (t *UI) removeCol(col *Column) {
	sentinel := &Column{nextCol: t.firstCol}
	prev := sentinel
	for prev.nextCol != nil {
		if prev.nextCol == col {
			prev.nextCol = col.nextCol
			col.nextCol = nil
			t.firstCol = sentinel.nextCol
			if t.firstCol != nil {
				t.firstCol.x = 0
			}
			return
		}
		prev = prev.nextCol
	}
	panic("column not found")
}

type Column struct {
	ui *UI
	x  int

	firstWin *Window
	nextCol  *Column
}

func (col *Column) handleMouseEvent(ev mouse.Event) {
	x, y := int(ev.X), int(ev.Y)

	if col.ui.grabbedCol != nil {
		if ev.Direction == mouse.DirRelease {
			col.ui.moveGrabbedCol(x)
		}
		return
	} else if col.ui.grabbedWin != nil {
		if ev.Direction == mouse.DirRelease {
			col.moveGrabbedWin(y - col.y())
		}
		return
	}

	if ev.Direction == mouse.DirPress && x == col.x && y == col.ui.y {
		col.ui.grabbedCol = col
		return
	}

	win := col.firstWin
	for win != nil {
		if y < win.y || y >= win.y+win.height() {
			win = win.nextWin
			continue
		}
		if y >= win.body.y {
			win.body.click(ev)
			col.ui.activeText = win.body
		} else {
			if ev.Direction == mouse.DirPress && x == win.col.x && y == win.tag.y {
				col.ui.grabbedWin = win
				break
			}
			win.tag.click(ev)
			col.ui.activeText = win.tag
		}
		break
	}
}

func (col *Column) newWindow() *Window {
	tag := &Text{
		frame:   new(Frame),
		bgstyle: tagbg,
		hlstyle: taghl,
	}
	body := &Text{
		frame:   new(Frame),
		bgstyle: bodybg,
		hlstyle: bodyhl,
	}
	win := &Window{
		col:  col,
		y:    0,
		tag:  tag,
		body: body,
	}
	tag.win = win
	body.win = win

	if col.firstWin == nil {
		if col.ui.activeText == nil {
			col.ui.activeText = body
		}
		col.firstWin = win
	} else {
		prev := col.lastWin()
		win.y = prev.y + prev.height()/2
		prev.nextWin = win
	}
	return win
}

func (col *Column) deleteWindow(todel *Window) {
	sentinel := &Window{nextWin: col.firstWin}
	win := sentinel
	for win.nextWin != nil {
		if win.nextWin == todel {
			win.nextWin = todel.nextWin
			col.firstWin = sentinel.nextWin
			if col.firstWin != nil {
				col.firstWin.y = 0
			}
			return
		}
		win = win.nextWin
	}
	panic("window not found")
}

func (col *Column) flush() {
	col.ui.screen.SetContent(col.x, col.ui.y, ' ', nil, testbg)
	for x := col.x + 1; x < col.x+col.width(); x++ {
		col.ui.screen.SetContent(x, col.ui.y, ' ', nil, tagbg)
	}
	if col.firstWin == nil {
		for x := col.x; x < col.x+col.width(); x++ {
			coly, colh := col.y(), col.height()
			for y := coly; y < coly+colh; y++ {
				col.ui.screen.SetContent(x, y, ' ', nil, whitebg)
			}
		}
		return
	}
	win := col.firstWin
	for win != nil {
		win.flush()
		win = win.nextWin
	}
}

func (col *Column) lastWin() *Window {
	win := col.firstWin
	if win == nil {
		return nil
	}
	for win.nextWin != nil {
		win = win.nextWin
	}
	return win
}

func (col *Column) moveGrabbedWin(y int) {
	gw := col.ui.grabbedWin
	col.ui.grabbedWin = nil
	target := col.firstWin

	if target == nil {
		gw.col.removeWin(gw)
		col.firstWin = gw
		gw.col = col
		gw.y = 0
		return
	}

	for target != nil {
		if y >= target.y && y < target.y+target.height() {
			break
		}
		target = target.nextWin
	}

	if y == target.y {
		// TODO: If this happens, adjust position of the windows
		// to ensure at least one line of each window is shown.
		// Forbid it for now as it would cause panic otherwise.
		return
	}

	if gw == target || (target.nextWin != nil && gw == target.nextWin) {
		if gw == col.firstWin {
			return
		}
	} else {
		gw.col.removeWin(gw)
		gw.col = col
		gw.nextWin = target.nextWin
		target.nextWin = gw
	}
	gw.y = y
}

func (col *Column) removeWin(win *Window) {
	sentinel := &Window{nextWin: col.firstWin}
	prev := sentinel
	for prev.nextWin != nil {
		if prev.nextWin == win {
			prev.nextWin = win.nextWin
			win.nextWin = nil
			col.firstWin = sentinel.nextWin
			if col.firstWin != nil {
				col.firstWin.y = 0
			}
			return
		}
		prev = prev.nextWin
	}
	panic("window not found")
}

func (col *Column) width() int {
	if col.nextCol == nil {
		return col.ui.width - col.x
	}
	return col.nextCol.x - col.x
}

// Column's content y and height.
func (col *Column) y() int      { return col.ui.y + 1 } // TODO: Replace 1 with the number of tag lines.
func (col *Column) height() int { return col.ui.height - col.y() }

type Window struct {
	col *Column

	y int

	tag  *Text
	body *Text

	dirty bool

	nextWin *Window
}

func (win *Window) Tag() *Text  { return win.tag }
func (win *Window) Body() *Text { return win.body }

func (win *Window) Clear() {
	w := win.col.width()
	h := win.height()
	win.tag.width = w - 1
	win.tag.height = h - 1
	win.tag.clear()

	win.body.width = w - 1
	win.body.height = h
	win.body.clear()
}

func (win *Window) SetDirty(dirty bool) {
	win.dirty = dirty
}

func (win *Window) Delete() {
	win.col.deleteWindow(win)
}

func (win *Window) flush() {
	winy := win.y + win.col.y()
	win.tag.x = win.col.x + 1
	win.tag.y = winy
	win.tag.flush()

	h := len(win.tag.frame.lines)
	win.tag.height = h

	y := 0
	for ; y < h; y++ {
		bg := win.tag.bgstyle
		if y == 0 && win.dirty {
			bg = dirtystyle
		}
		win.col.ui.screen.SetContent(win.col.x, winy+y, ' ', nil, bg)
	}
	winh := win.height()
	for ; y < winh; y++ {
		win.col.ui.screen.SetContent(win.col.x, winy+y, ' ', nil, borderstyle)
	}

	win.body.height = winh - h
	if len(win.body.frame.lines) > win.body.height {
		// TODO: We didn't know how many lines will the tag of the window
		// span. Can we do better?
		win.body.frame.lines = win.body.frame.lines[:win.body.height]
	}
	win.body.x = win.col.x + 1
	win.body.y = winy + h
	win.body.flush()
	win.body.fill()
}

func (win *Window) height() int {
	if win.nextWin == nil {
		return win.col.height() - win.y
	}
	return win.nextWin.y - win.y
}

type Text struct {
	win   *Window
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

	mouseEventHandler ui.MouseEventHandler
	keyEventHandler   ui.KeyEventHandler
}

// TODO: Probably remove.
func (t *Text) Size() (w, h int) {
	return t.width, t.height
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
			w := 1
			switch {
			case r == '\n':
				goto fill
			case r == '\t':
				r = ' '
				w = tabWidthForCol(x)
			case r == 0:
				// TODO: This is a workaround to print silently \0 that
				// separates filename and commands in the tag of the window.
				r = ' '
			case !unicode.IsPrint(r):
				r = utf8.RuneError
			}
			for i := 0; i < w && x < t.width; i++ {
				// TODO: Should the rest of the tab at the end of a
				// line span the begining of the next line?
				t.win.col.ui.screen.SetContent(t.x+x, t.y+y, r, nil, style)
				x++
				if style == reverse {
					style = t.bgstyle
				}
			}
		}
		selStyle(p)
	fill:
		for ; x < t.width; x++ {
			t.win.col.ui.screen.SetContent(t.x+x, t.y+y, ' ', nil, style)
			if style == reverse {
				style = t.bgstyle
			}
		}
	}
}

func (t *Text) fill() {
	// TODO: Using this bg color just for testing purposes.
	bg := testbg
	for y := len(t.frame.lines); y < t.height; y++ {
		for x := 0; x < t.width; x++ {
			t.win.col.ui.screen.SetContent(t.x+x, t.y+y, ' ', nil, bg)
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
