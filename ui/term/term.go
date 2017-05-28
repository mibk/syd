package term

import (
	"fmt"
	"io"
	"sort"
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
		ui:      t,
		parent:  t,
		frame:   new(Frame),
		bgstyle: tagbg,
		hlstyle: taghl,
	}
	t.tag.init(t.model.Tag())

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
		t.reload()
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

func (t *UI) SetTag(m ui.Model) { t.tag.init(m) }

func (t *UI) reload() error {
	t.clear()
	t.tag.reload()
	col := t.firstCol
	for col != nil {
		if err := col.reload(); err != nil {
			return nil
		}
		col = col.next
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
		col = col.next
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
		if int(ev.X) < col.x()+col.width() {
			col.handleMouseEvent(ev)
			return
		}
		col = col.next
	}
}

func (t *UI) Push_Key_Event(ev key.Event) {
}

func (t *UI) NewColumn(m ui.Model) ui.Column {
	model := m.(*core.Column)
	tag := &Text{
		ui:      t,
		frame:   new(Frame),
		bgstyle: tagbg,
		hlstyle: taghl,
	}
	tag.init(model.Tag())
	col := &Column{
		ui:    t,
		model: model,
		tag:   tag,
	}
	tag.parent = col
	if t.firstCol == nil {
		t.firstCol = col
	} else {
		prev := t.lastCol()
		col.setx(prev.x() + prev.width()*3/5)
		prev.next = col
	}
	return col
}

func (t *UI) lastCol() *Column {
	col := t.firstCol
	if col == nil {
		return nil
	}
	for col.next != nil {
		col = col.next
	}
	return col
}

func (t *UI) moveGrabbedCol(x int) {
	gc := t.grabbedCol
	t.grabbedCol = nil

	t.model.MoveColumn(gc.model, float64(x)/float64(t.width))

	// TODO: Just a temporary hack.
	var cols []*Column
	col := t.firstCol
	for col != nil {
		cols = append(cols, col)
		col = col.next
	}
	sort.Slice(cols, func(i, j int) bool {
		return cols[i].model.X() < cols[j].model.X()
	})

	sentinel := &Column{}
	prev := sentinel
	for _, col := range cols {
		prev.next = col
		prev = col
	}
	prev.next = nil
	t.firstCol = sentinel.next
}

func (t *UI) removeCol(col *Column) {
	sentinel := &Column{next: t.firstCol}
	prev := sentinel
	for prev.next != nil {
		if prev.next == col {
			prev.next = col.next
			col.next = nil
			t.firstCol = sentinel.next
			if t.firstCol != nil {
				t.firstCol.setx(0)
			}
			return
		}
		prev = prev.next
	}
	panic("column not found")
}

func (t *UI) y() int {
	return ui_y + len(t.tag.frame.lines)
}

type Column struct {
	ui    *UI
	model *core.Column

	tag *Text

	firstWin *Window
	next     *Column
}

func (col *Column) SetTag(m ui.Model) { col.tag.init(m) }

func (col *Column) x() int {
	return int(col.model.X() * float64(col.ui.width))
}

func (col *Column) setx(x int) {
	col.model.SetX(float64(x) / float64(col.ui.width))
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

	if ev.Direction == mouse.DirPress && x == col.x() && y == col.ui.y() {
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
		if winY := win.y(); y < winY || y >= winY+win.height() {
			win = win.next
			continue
		}
		if y >= win.body.y {
			win.body.handleMouseEvent(ev)
			col.ui.activeText = win.body
		} else {
			if ev.Direction == mouse.DirPress && x == win.col.x() && y == win.tag.y {
				col.ui.grabbedWin = win
				break
			}
			win.tag.handleMouseEvent(ev)
			col.ui.activeText = win.tag
		}
		break
	}
}

func (col *Column) NewWindow(m ui.Model) ui.Updater {
	model := m.(*core.Window)
	tag := &Text{
		ui:      col.ui,
		frame:   new(Frame),
		bgstyle: tagbg,
		hlstyle: taghl,
	}
	body := &Text{
		ui:      col.ui,
		frame:   new(Frame),
		bgstyle: bodybg,
		hlstyle: bodyhl,
	}
	tag.init(model.Tag())
	body.init(model.Body())
	win := &Window{
		model: model,
		col:   col,
		tag:   tag,
		body:  body,
	}
	tag.parent = win
	body.parent = win

	if col.firstWin == nil {
		if col.ui.activeText == nil {
			col.ui.activeText = body
		}
		col.firstWin = win
	} else {
		prev := col.lastWin()
		win.sety(prev.y() + prev.height()/2)
		prev.next = win
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
		win = win.next
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
	col.tag.x = col.x() + 1
	col.tag.y = uiy
	col.tag.flush()

	col.ui.screen.SetContent(col.x(), uiy, ' ', nil, testbg)
	for y := uiy + 1; y < col.y(); y++ {
		col.ui.screen.SetContent(col.x(), y, ' ', nil, col.tag.bgstyle)
	}

	if col.firstWin == nil {
		for x := col.x(); x < col.x()+col.width(); x++ {
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
		win = win.next
	}
}

func (col *Column) lastWin() *Window {
	win := col.firstWin
	if win == nil {
		return nil
	}
	for win.next != nil {
		win = win.next
	}
	return win
}

func (col *Column) moveGrabbedWin(y int) {
	gw := col.ui.grabbedWin
	col.ui.grabbedWin = nil

	col.model.MoveWindow(gw.model, float64(y)/float64(col.ui.height))

	// TODO: Just a temporary hack.
	var wins []*Window
	win := col.firstWin
	for win != nil {
		wins = append(wins, win)
		win = win.next
	}
	sort.Slice(wins, func(i, j int) bool {
		return wins[i].model.Y() < wins[j].model.Y()
	})

	sentinel := &Window{}
	prev := sentinel
	for _, win := range wins {
		prev.next = win
		prev = win
	}
	prev.next = nil
	col.firstWin = sentinel.next
}

func (col *Column) removeWin(win *Window) {
	sentinel := &Window{next: col.firstWin}
	prev := sentinel
	for prev.next != nil {
		if prev.next == win {
			prev.next = win.next
			win.next = nil
			col.firstWin = sentinel.next
			if col.firstWin != nil {
				col.firstWin.sety(0)
			}
			return
		}
		prev = prev.next
	}
	panic("window not found")
}

func (col *Column) width() int {
	if col.next == nil {
		return col.ui.width - col.x()
	}
	return col.next.x() - col.x()
}

// Column's content y and height.
func (col *Column) y() int      { return col.ui.y() + len(col.tag.frame.lines) }
func (col *Column) height() int { return col.ui.height - col.y() }

type Window struct {
	col   *Column
	model *core.Window

	tag  *Text
	body *Text

	next *Window
}

func (win *Window) y() int {
	return int(win.model.Y() * float64(win.col.ui.height))
}

func (win *Window) sety(y int) {
	win.model.SetY(float64(y) / float64(win.col.ui.height))
}

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
	winy := win.y() + win.col.y()
	win.tag.x = win.col.x() + 1
	win.tag.y = winy
	win.tag.flush()

	y := 0
	for ; y < h; y++ {
		bg := win.tag.bgstyle
		if y == 0 && win.model.Dirty() {
			bg = dirtystyle
		}
		win.col.ui.screen.SetContent(win.col.x(), winy+y, ' ', nil, bg)
	}
	winh := win.height()
	for ; y < winh; y++ {
		win.col.ui.screen.SetContent(win.col.x(), winy+y, ' ', nil, borderstyle)
	}

	win.body.height = winh - h
	if len(win.body.frame.lines) > win.body.height {
		// TODO: We didn't know how many lines will the tag of the window
		// span. Can we do better?
		win.body.frame.lines = win.body.frame.lines[:win.body.height]
	}
	win.body.x = win.col.x() + 1
	win.body.y = winy + h
	win.body.flush()
	win.body.fill()
}

func (win *Window) height() int {
	if win.next == nil {
		return win.col.height() - win.y()
	}
	return win.next.y() - win.y()
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

func (t *Text) init(m ui.Model) {
	// TODO: Come up with a better design.
	t.model = m.(*core.Text)
}

func (t *Text) handleKeyEvent(ev key.Event) {
	switch {
	case ev.Rune == ui.KeyEnter:
		t.model.InsertNewLine()
		t.checkVisibility()
	case ev.Rune == ui.KeyBackspace:
		q0, q1 := t.model.Selected()
		if q0 == q1 {
			t.sel(q0-1, q1)
		}
		t.deleteSel()
	case ev.Rune == ui.KeyDelete:
		q0, q1 := t.model.Selected()
		if q0 == q1 {
			t.sel(q0, q1+1)
		}
		t.deleteSel()
	case ev.Rune == ui.KeyEscape:
		t.deleteSel()
	case ev.Rune == ui.KeyLeft:
		left(t)
	case ev.Rune == ui.KeyRight:
		right(t)

	case ev.Rune == ui.KeyUp:
		t.up()
	case ev.Rune == ui.KeyDown:
		t.down()

	case (ev.Rune == 'c' || ev.Rune == 'x') && ev.Modifiers&key.ModControl != 0:
		s := t.model.SelectionToString(t.model.Selected())
		err := clipboard.WriteAll(s)
		if err != nil {
			panic(err)
		}
		if ev.Rune == 'x' {
			t.deleteSel()
		}
	case ev.Rune == 'v' && ev.Modifiers&key.ModControl != 0:
		s, err := clipboard.ReadAll()
		if err != nil {
			panic(err)
		}
		t.insert(s)
	default:
		t.insert(string(ev.Rune))
	}
}

func (t *Text) sel(q0, q1 int64) {
	t.model.Select(q0, q1)
	t.checkVisibility()
}

func (t *Text) insert(s string) {
	t.model.Insert(s)
	t.frame.SetWantCol(ui.ColQ1)
	t.checkVisibility()
}

func (t *Text) deleteSel() {
	t.model.DeleteSel()
	t.checkVisibility()
}

func (t *Text) checkVisibility() {
	t.reloadWithParent()
	origin := t.model.Origin()
	q0, _ := t.model.Selected()
	if q0 < origin || q0 > origin+int64(t.frame.Nchars())+1 {
		t.model.SetOrigin(t.model.PrevNewLine(q0, 3))
	}
}

func left(t *Text) {
	q0, _ := t.model.Selected()
	t.sel(q0-1, q0-1)
	t.frame.SetWantCol(ui.ColQ0)
}

func right(t *Text) {
	_, q1 := t.model.Selected()
	t.sel(q1+1, q1+1)
	t.frame.SetWantCol(ui.ColQ1)
}

func (t *Text) up() {
	_, line1 := t.frame.SelectionLines()
	q := t.findQ(line1 - 1)
	t.sel(q, q)
}

func (t *Text) down() {
	_, line1 := t.frame.SelectionLines()
	q := t.findQ(line1 + 1)
	t.sel(q, q)
}

func (t *Text) findQ(line int) int64 {
	if line < 0 {
		t.model.SetOrigin(t.model.PrevNewLine(t.model.Origin(), -line))
		t.reloadWithParent()
		line = 0
	} else if line > t.frame.Lines()-1 {
		if t.frame.Lines() == t.height {
			i := line - t.frame.Lines() + 1
			oldOrg := t.model.Origin()
			l := t.frame.Lines()
			t.model.SetOrigin(oldOrg + int64(t.frame.CharsUntilXY(0, i)))
			t.reloadWithParent()
			if t.frame.Lines() < l {
				t.model.SetOrigin(oldOrg)
				t.reloadWithParent()
			}
		}
		line = t.frame.Lines() - 1
	}
	q := t.model.Origin()
	return q + int64(t.frame.CharsUntilXY(t.frame.WantCol(), line))
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
			t.frame.SetWantCol(ui.ColQ0)
		}
	case mouse.DirRelease:
		t.model.StopSel()
	case mouse.DirNone:
		t.model.MoveSel(q)
	case mouse.DirStep:
		const nlines = 3
		switch ev.Button {
		case mouse.ButtonWheelUp:
			neworg := t.model.PrevNewLine(t.model.Origin(), nlines)
			t.model.SetOrigin(neworg)
		case mouse.ButtonWheelDown:
			n := int64(t.frame.CharsUntilXY(0, nlines))
			t.model.SetOrigin(t.model.Origin() + n)
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

func (t *Text) reloadWithParent() error { return t.parent.reload() }

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
