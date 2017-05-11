package core

import (
	"os"

	"github.com/mibk/syd/ui"
	"github.com/mibk/syd/undo"
)

type Column struct {
	ed   *Editor
	tag  *Text
	wins []*Window
	col  ui.Column
}

func (col *Column) redraw() {
	col.tag.redraw()
	for _, win := range col.wins {
		win.redraw()
	}
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
	window := col.col.NewWindow()
	buf := NewUndoBuffer(undo.NewBuffer(con.Bytes()))
	win := &Window{col: col, win: window, con: con, buf: buf}
	win.tag = newText(win, &BasicBuffer{[]rune("\x00Del Put Undo Redo ")}, window.Tag())
	win.body = newText(win, buf, window.Body())
	col.wins = append(col.wins, win)
	return win
}

func (col *Column) deleteWindow(todel *Window) {
	for i, win := range col.wins {
		if win == todel {
			col.wins = append(col.wins[:i], col.wins[i+1:]...)
			return
		}
	}
	panic("window not found")
}

func (col *Column) removeWindow(todel ui.Window) *Window {
	for i, win := range col.wins {
		if win.win == todel {
			col.wins = append(col.wins[:i], col.wins[i+1:]...)
			return win
		}
	}
	panic("window not found")
}

func (col *Column) Close() error {
	for _, win := range col.wins {
		// TODO: Check errors.
		win.Close()
	}
	col.col.Delete()
	col.ed.deleteColumn(col)
	return nil
}

func (col *Column) editor() (ed *Editor)         { return col.ed }
func (col *Column) column() (c *Column, ok bool) { return col, true }
func (col *Column) window() (w *Window, ok bool) { return nil, false }
