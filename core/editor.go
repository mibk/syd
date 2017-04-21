package core

import (
	"github.com/mibk/syd/ui"
	"github.com/mibk/syd/ui/term"
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
	wins map[string]*Window
	mode int
}

func NewEditor(u *term.UI) *Editor {
	ed := &Editor{
		ui:     u,
		events: make(chan ui.Event),
		wins:   make(map[string]*Window),
	}
	// TODO: Move the cursor to the end of the line.
	ed.tag = newText(ed, &BasicBuffer{[]rune("Newcol Exit ")}, u.Tag())
	q := ed.tag.buf.End()
	ed.tag.q0, ed.tag.q1 = q, q
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
	q := col.tag.buf.End()
	col.tag.q0, col.tag.q1 = q, q
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
