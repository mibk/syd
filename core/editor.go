package core

import (
	"os"

	"github.com/mibk/syd/ui"
	"github.com/mibk/syd/ui/term"
	"github.com/mibk/syd/undo"
	"github.com/mibk/syd/vi"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/mouse"
)

type Editor struct {
	ui         *term.UI
	events     chan ui.Event
	vi         *vi.Parser
	shouldQuit bool

	tag *Text

	cols []*Column
	mode int
}

func NewEditor(u *term.UI) *Editor {
	ed := &Editor{
		ui:     u,
		events: make(chan ui.Event),
	}
	ed.tag = newText(ed, &BasicBuffer{[]rune("Newcol Exit ")}, u.Tag())
	return ed
}

func (ed *Editor) Main() {
	for !ed.shouldQuit {
		ed.ui.Clear()
		ed.tag.loadText()
		for _, col := range ed.cols {
			col.Refresh()
		}
		ed.ui.Flush()
		ev := <-ui.Events
		if ev == ui.Quit {
			return
		}
		switch ev := ev.(type) {
		case key.Event:
			ed.ui.Push_Key_Event(ev)
		case mouse.Event:
			// Temporary reasons...
			ed.ui.Push_Mouse_Event(ev)
		}
	}
}

func (ed *Editor) NewColumn() *Column {
	col := &Column{
		ed:  ed,
		col: ed.ui.NewColumn(),
	}
	ed.cols = append(ed.cols, col)
	col.tag = newText(col, &BasicBuffer{[]rune("New Delcol ")}, col.col.Tag())
	col.col.OnWindowMoved(func(win *term.Window, from *term.Column) {
		if from == col.col {
			return
		}
		fromCol := ed.findColumn(from)
		ww := fromCol.removeWindow(win)
		ww.col = col
		col.wins = append(col.wins, ww)
	})
	return col
}

func (ed *Editor) Close() error {
	for _, col := range ed.cols {
		col.Close()
	}
	return nil
}

func (ed *Editor) findColumn(tofind *term.Column) *Column {
	for _, col := range ed.cols {
		if col.col == tofind {
			return col
		}
	}
	panic("column not found")
}

func (ed *Editor) deleteColumn(todel *Column) {
	for i, col := range ed.cols {
		if col == todel {
			ed.cols = append(ed.cols[:i], ed.cols[i+1:]...)
			return
		}
	}
	panic("column not found")
}

func (ed *Editor) editor() *Editor         { return ed }
func (ed *Editor) column() (*Column, bool) { return nil, false }
func (ed *Editor) window() (*Window, bool) { return nil, false }

type Column struct {
	ed   *Editor
	tag  *Text
	wins []*Window
	col  *term.Column
}

func (col *Column) Refresh() {
	col.col.Clear()
	col.tag.loadText()
	for _, win := range col.wins {
		win.LoadText()
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
			// TODO: Once columns and editor tags are implemented,
			// remove this.
			for _, col := range col.ed.cols {
				if len(col.wins) > 0 {
					return
				}
			}
			col.ed.shouldQuit = true
			return
		}
	}
	panic("window not found")
}

func (col *Column) removeWindow(todel *term.Window) *Window {
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
