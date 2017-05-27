package core

import (
	"os"

	"github.com/mibk/syd/ui"
	"github.com/mibk/syd/undo"
)

type Column struct {
	ed       *Editor
	tag      *Text
	firstWin *Window
	col      ui.Column

	x float64

	next *Column
}

func (col *Column) NewWindow() *Window {
	return col.newWindow(BytesContent([]byte{}))
}

func (col *Column) NewWindowFile(filename string) (*Window, error) {
	f, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			win := col.NewWindow()
			win.SetFilename(filename)
			return win, nil
		}
		return nil, err
	}
	mm, err := Mmap(f)
	if err != nil {
		return nil, err
	}
	win := col.newWindow(mm)
	win.SetFilename(filename)
	q := win.tag.buf.End()
	win.tag.q0, win.tag.q1 = q, q
	return win, nil
}

func (col *Column) newWindow(con Content) *Window {
	buf := NewUndoBuffer(undo.NewBuffer(con.Bytes()))
	win := &Window{con: con, buf: buf}
	window := col.col.NewWindow(win)
	win.win = window
	win.tag = newText(win, &BasicBuffer{[]rune("\x00Del Put Undo Redo ")}, window.Tag())
	win.body = newText(win, buf, window.Body())
	col.appendWindow(win)
	return win
}

func (col *Column) appendWindow(win *Window) {
	win.col = col
	win.next = nil
	sentinel := &Window{next: col.firstWin}
	prev := sentinel
	for prev.next != nil {
		prev = prev.next
	}
	prev.next = win
	col.firstWin = sentinel.next
}

func (col *Column) removeWindow(todel *Window) {
	sentinel := &Window{next: col.firstWin}
	win := sentinel
	for win.next != nil {
		if win.next == todel {
			win.next = todel.next
			todel.next = nil
			col.firstWin = sentinel.next
			if col.firstWin != nil {
				col.firstWin.SetY(0)
			}
			return
		}
		win = win.next
	}
	panic("window not found")
}

func (col *Column) X() float64 { return col.x }

func (col *Column) right() float64 {
	if col.next == nil {
		return 1
	}
	return col.next.x
}

func (col *Column) SetX(x float64) {
	if x < 0 || x > 1 {
		panic("x must be in the range 0..1")
	}
	col.x = x
}

func (col *Column) MoveWindow(win *Window, y float64) {
	if col.firstWin == nil {
		col.maybe_Move_To_Different_Column(win)
		win.col.removeWindow(win)
		col.firstWin = win
		win.col = col
		win.SetY(0)
		return
	}

	target := col.firstWin
	for target != nil {
		if y < target.bottom() {
			break
		}
		target = target.next
	}
	if target == nil {
		return
	}

	if y == target.y {
		// TODO: If this happens, adjust position of the windows
		// to ensure at least one line of each window is shown.
		// Forbid it for now as it would cause panic otherwise.
		return
	}

	if win == target || (target.next != nil && win == target.next) {
		if win == col.firstWin {
			return
		}
	} else {
		col.maybe_Move_To_Different_Column(win)
		win.col.removeWindow(win)
		win.col = col
		win.next = target.next
		target.next = win
	}
	win.SetY(y)
}

// TODO: Temporary hack.
func (col *Column) maybe_Move_To_Different_Column(win *Window) {
	if col != win.col {
		win.win.Update(ui.Delete)
		ww := col.col.NewWindow(win)
		ww.Tag().Init(win.tag)
		ww.Body().Init(win.body)
		win.win = ww
	}
}

func (col *Column) Close() error {
	win := col.firstWin
	for win != nil {
		// TODO: Check errors.
		win.Close()
		win = win.next
	}
	col.col.Update(ui.Delete)
	col.ed.removeColumn(col)
	return nil
}

func (col *Column) editor() (ed *Editor)         { return col.ed }
func (col *Column) column() (c *Column, ok bool) { return col, true }
func (col *Column) window() (w *Window, ok bool) { return nil, false }
