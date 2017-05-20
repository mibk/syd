package term

import (
	"fmt"
	"io"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/mouse"

	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell"
	"github.com/mibk/syd/core"
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

type reloader interface {
	reload() error
}

type UI struct {
	screen tcell.Screen
	model  *core.Editor

	wasBtnPressed bool

	width  int
	height int

	tag      *Text
	firstCol *Column

	grabbedCol *Column // grabbed col or nil
	grabbedWin *Window // grabbed win or nil
	activeText *Text   // will receive key events
}

func (t *UI) Init(m ui.Model) error {
	t.model = m.(*core.Editor)
	sc, err := tcell.NewScreen()
	if err != nil {
		return err
	}
	if err := sc.Init(); err != nil {
		return err
	}
	sc.EnableMouse()
	t.screen = sc

	t.tag = &Text{
		parent:  t,
		frame:   new(Frame),
		bgstyle: tagbg,
		hlstyle: taghl,
	}
	t.tag.ui = t

	// TODO: Just for testing purposes.
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

func (t *UI) Main() {
	for {
		t.model.Refresh()
		t.flush()
		ev := <-ui.Events
		if ev == ui.Quit {
			return
		}
		switch ev := ev.(type) {
		case key.Event:
			t.activeText.handleKeyEvent(ev)
		case mouse.Event:
			t.handleMouseEvent(ev)
		}
	}
}

func (t *UI) Size() (w, h int) { return t.screen.Size() }

func (t *UI) Tag() ui.Text { return t.tag }

func (t *UI) reload() error {
	t.clear()
	t.tag.reload()
	col := t.firstCol
	for col != nil {
		if err := col.reload(); err != nil {
			return nil
		}
		col = col.nextCol
	}
	return nil
}

func (t *UI) clear() {
	t.tag.width = t.width - 1
	t.tag.height = t.height - 1
	t.tag.clear()
}

// TODO: Just for testing purposes; remove.
const ui_y = 1

func (t *UI) flush() {
	t.tag.x = ui_y
	t.tag.y = ui_y
	t.tag.height = len(t.tag.frame.lines)
	t.tag.flush()

	for y := ui_y; y < ui_y+t.tag.height; y++ {
		t.screen.SetContent(0, y, ' ', nil, t.tag.bgstyle)
	}

	col := t.firstCol
	for col != nil {
		col.flush()
		col = col.nextCol
	}

	if t.firstCol == nil {
		uiy := t.y()
		for x := 0; x < t.width; x++ {
			for y := uiy; y < uiy+t.height; y++ {
				t.screen.SetContent(x, y, ' ', nil, whitebg)
			}
		}
	}

	t.screen.Show()
}

func (t *UI) handleMouseEvent(ev mouse.Event) {
	y := int(ev.Y)
	if y < t.y() {
		t.tag.handleMouseEvent(ev)
		t.activeText = t.tag
		return
	}

	col := t.firstCol
	for col != nil {
		if int(ev.X) < col.x+col.width() {
			col.handleMouseEvent(ev)
			return
		}
		col = col.nextCol
	}
}

func (t *UI) Push_Key_Event(ev key.Event) {
}

func (t *UI) NewColumn(m ui.Model) ui.Column {
	model := m.(*core.Column)
	tag := &Text{
		frame:   new(Frame),
		bgstyle: tagbg,
		hlstyle: taghl,
	}
	col := &Column{
		ui:    t,
		model: model,
		tag:   tag,
	}
	tag.ui = t
	tag.parent = col
	if t.firstCol == nil {
		t.firstCol = col
	} else {
		prev := t.lastCol()
		col.x = prev.x + prev.width()*3/5
		prev.nextCol = col
	}
	return col
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
	if target == nil {
		// Nothing to do.
		return
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

func (t *UI) y() int {
	return ui_y + len(t.tag.frame.lines)
}

type Column struct {
	ui    *UI
	model *core.Column

	x int

	tag *Text

	firstWin *Window
	nextCol  *Column
}

func (col *Column) Tag() ui.Text { return col.tag }

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

	if ev.Direction == mouse.DirPress && x == col.x && y == col.ui.y() {
		col.ui.grabbedCol = col
		return
	}

	if y >= col.tag.y && y < col.y() {
		col.tag.handleMouseEvent(ev)
		col.ui.activeText = col.tag
		return
	}

	win := col.firstWin
	for win != nil {
		if y < win.y || y >= win.y+win.height() {
			win = win.nextWin
			continue
		}
		if y >= win.body.y {
			win.body.handleMouseEvent(ev)
			col.ui.activeText = win.body
		} else {
			if ev.Direction == mouse.DirPress && x == win.col.x && y == win.tag.y {
				col.ui.grabbedWin = win
				break
			}
			win.tag.handleMouseEvent(ev)
			col.ui.activeText = win.tag
		}
		break
	}
}

func (col *Column) NewWindow(m ui.Model) ui.Window {
	model := m.(*core.Window)
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
		model: model,
		col:   col,
		y:     0,
		tag:   tag,
		body:  body,
	}
	tag.ui = col.ui
	tag.parent = win
	body.ui = col.ui
	body.parent = win

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

func (col *Column) Update(msg ui.Message) {
	switch msg {
	case ui.Delete:
		col.ui.removeCol(col)
	default:
		panic(fmt.Sprintf("unexpected message: %v", msg))
	}
}

func (col *Column) reload() error {
	col.clear()
	col.tag.reload()
	win := col.firstWin
	for win != nil {
		if err := win.reload(); err != nil {
			return err
		}
		win = win.nextWin
	}
	return nil
}

func (col *Column) clear() {
	w := col.width()
	h := col.height()
	col.tag.width = w - 1
	col.tag.height = h - 1
	col.tag.clear()
}

func (col *Column) flush() {
	uiy := col.ui.y()
	h := len(col.tag.frame.lines)
	col.tag.height = h
	col.tag.x = col.x + 1
	col.tag.y = uiy
	col.tag.flush()

	col.ui.screen.SetContent(col.x, uiy, ' ', nil, testbg)
	for y := uiy + 1; y < col.y(); y++ {
		col.ui.screen.SetContent(col.x, y, ' ', nil, col.tag.bgstyle)
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

	if col.firstWin == nil {
		gw.model.MoveToColumn(col.model)
		gw.col.removeWin(gw)
		col.firstWin = gw
		gw.col = col
		gw.y = 0
		return
	}

	target := col.firstWin
	for y < target.y || y >= target.y+target.height() {
		target = target.nextWin
		if target == nil {
			// Nothing to do.
			return
		}
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
		gw.model.MoveToColumn(col.model)
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
func (col *Column) y() int      { return col.ui.y() + len(col.tag.frame.lines) }
func (col *Column) height() int { return col.ui.height - col.y() }

type Window struct {
	col   *Column
	model *core.Window

	y int

	tag  *Text
	body *Text

	nextWin *Window
}

func (win *Window) Tag() ui.Text  { return win.tag }
func (win *Window) Body() ui.Text { return win.body }

func (win *Window) reload() error {
	win.clear()
	if err := win.tag.reload(); err != nil {
		return err
	}
	if err := win.body.reload(); err != nil {
		return err
	}
	return nil
}

func (win *Window) clear() {
	w := win.col.width()
	h := win.height()
	win.tag.width = w - 1
	win.tag.height = h - 1
	win.tag.clear()

	win.body.width = w - 1
	win.body.height = h
	win.body.clear()
}

func (win *Window) Update(msg ui.Message) {
	switch msg {
	case ui.Delete:
		win.col.removeWin(win)
	default:
		panic(fmt.Sprintf("unexpected message: %v", msg))
	}
}

func (win *Window) flush() {
	h := len(win.tag.frame.lines)
	win.tag.height = h
	winy := win.y + win.col.y()
	win.tag.x = win.col.x + 1
	win.tag.y = winy
	win.tag.flush()

	y := 0
	for ; y < h; y++ {
		bg := win.tag.bgstyle
		if y == 0 && win.model.Dirty() {
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
	ui    *UI
	frame *Frame

	model *core.Text

	// TODO: If text is reloaded, it might require reloading
	// of the parent component (e.g. when Text represents tag
	// and the tag changes the number of lines). Revalidate
	// whether this is still true once the ui API settles down.
	parent reloader

	width, height int
	x, y          int

	cur struct {
		p0, p1 int // char position
		x, y   int // current position
	}
	timestamp time.Time // last clicked

	// styles
	bgstyle tcell.Style
	hlstyle tcell.Style
}

// Init initializes t so it can be safely used.
func (t *Text) Init(m ui.Model) {
	// TODO: Come up with a better design.
	t.model = m.(*core.Text)
}

// TODO: Probably remove.
func (t *Text) Size() (w, h int) {
	return t.width, t.height
}

func (t *Text) handleKeyEvent(ev key.Event) {
	switch {
	case ev.Rune == ui.KeyEnter:
		t.model.InsertNewLine()
	case ev.Rune == ui.KeyBackspace:
		q0, q1 := t.model.Selected()
		if q0 == q1 {
			t.model.Select(q0-1, q1)
		}
		t.model.DeleteSel()
	case ev.Rune == ui.KeyDelete:
		q0, q1 := t.model.Selected()
		if q0 == q1 {
			t.model.Select(q0, q1+1)
		}
		t.model.DeleteSel()
	case ev.Rune == ui.KeyEscape:
		t.model.DeleteSel()
	case ev.Rune == ui.KeyLeft:
		left(t)
	case ev.Rune == ui.KeyRight:
		right(t)

	case ev.Rune == ui.KeyUp:
		t.model.Up()
	case ev.Rune == ui.KeyDown:
		t.model.Down()

	case (ev.Rune == 'c' || ev.Rune == 'x') && ev.Modifiers&key.ModControl != 0:
		s := t.model.SelectionToString(t.model.Selected())
		err := clipboard.WriteAll(s)
		if err != nil {
			panic(err)
		}
		if ev.Rune == 'x' {
			t.model.DeleteSel()
		}
	case ev.Rune == 'v' && ev.Modifiers&key.ModControl != 0:
		s, err := clipboard.ReadAll()
		if err != nil {
			panic(err)
		}
		t.model.Insert(s)
	default:
		t.model.Insert(string(ev.Rune))
	}
}

func left(t *Text) {
	q0, _ := t.model.Selected()
	t.model.Select(q0-1, q0-1)
	t.frame.SetWantCol(ui.ColQ0)
}

func right(t *Text) {
	_, q1 := t.model.Selected()
	t.model.Select(q1+1, q1+1)
	t.frame.SetWantCol(ui.ColQ1)
}

func (t *Text) handleMouseEvent(ev mouse.Event) {
	p := t.frame.CharsUntilXY(int(ev.X)-t.x, int(ev.Y)-t.y)
	q := t.model.Origin() + int64(p)

	switch ev.Direction {
	case mouse.DirPress:
		switch {
		case ev.Button == mouse.ButtonMiddle:
			t.model.ExecuteUnderCursor(q)
		case ev.Button == mouse.ButtonRight:
			t.model.Plumb(q)
		case time.Since(t.timestamp) < 300*time.Millisecond:
			t.model.SelectUnderCursor(q)
		default:
			t.timestamp = time.Now()
			t.model.StartSel(q)
		}
	case mouse.DirRelease:
		t.model.StopSel()
	case mouse.DirNone:
		t.model.MoveSel(q)
	case mouse.DirStep:
		switch ev.Button {
		case mouse.ButtonWheelUp:
			t.model.ScrollUp(3)
		case mouse.ButtonWheelDown:
			t.model.ScrollDown(3)
		}
	}
}

func (t *Text) clear() {
	*t.frame = Frame{
		lines:   make([][]rune, 1),
		wantCol: t.frame.wantCol,
	}
	t.cur.x, t.cur.y = 0, 0

	q0, q1 := t.model.Selected()
	origin := t.model.Origin()
	t.cur.p0, t.cur.p1 = int(q0-origin), int(q1-origin)

	t.checkSelection()
}

func (t *Text) Select(p0, p1 int) { t.cur.p0, t.cur.p1 = p0, p1 }

func (t *Text) Reload() error { return t.parent.reload() }

func (t *Text) reload() error {
	t.model.Reset()
	for {
		r, _, err := t.model.ReadRune()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if err := t.writeRune(r); err != nil {
			break
		}
	}
	return nil
}

func (t *Text) writeRune(r rune) error {
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
				t.ui.screen.SetContent(t.x+x, t.y+y, r, nil, style)
				x++
				if style == reverse {
					style = t.bgstyle
				}
			}
		}
		selStyle(p)
	fill:
		for ; x < t.width; x++ {
			t.ui.screen.SetContent(t.x+x, t.y+y, ' ', nil, style)
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
			t.ui.screen.SetContent(t.x+x, t.y+y, ' ', nil, bg)
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
