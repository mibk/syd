package core

import (
	"github.com/mibk/syd/ui"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/mouse"
)

type Editor struct {
	ui         ui.UI
	events     chan ui.Event
	shouldQuit bool

	tag *Text

	// TODO: Make it just an io.Reader so it's easier to work
	// with.
	errWin *Window

	cols []*Column
	wins map[string]*Window
	mode int
}

func NewEditor(u ui.UI) *Editor {
	ed := &Editor{
		ui:     u,
		events: make(chan ui.Event),
		wins:   make(map[string]*Window),
	}
	ed.tag = newText(ed, &BasicBuffer{[]rune("Newcol Exit ")}, u.Tag())
	q := ed.tag.buf.End()
	ed.tag.q0, ed.tag.q1 = q, q
	return ed
}

func (ed *Editor) Main() {
	for !ed.shouldQuit {
		ed.tag.redraw()
		for _, col := range ed.cols {
			col.redraw()
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
	col := &Column{ed: ed}
	ed.cols = append(ed.cols, col)

	column := ed.ui.NewColumn(col)
	col.col = column
	col.tag = newText(col, &BasicBuffer{[]rune("New Delcol ")}, col.col.Tag())
	q := col.tag.buf.End()
	col.tag.q0, col.tag.q1 = q, q
	return col
}

func (ed *Editor) Close() error {
	for len(ed.cols) > 0 {
		ed.cols[len(ed.cols)-1].Close()
	}
	return nil
}

func (ed *Editor) recentCol() *Column {
	if len(ed.cols) == 0 {
		return ed.NewColumn()
	}
	return ed.cols[0]
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

func (ed *Editor) stderr() writeFlusher {
	return &outputWriter{ed: ed}
}

type outputWriter struct {
	ed    *Editor
	ready bool
}

func (w *outputWriter) Write(b []byte) (n int, err error) {
	ed := w.ed
	if !w.ready {
		w.ready = true
		if ed.errWin == nil {
			ed.errWin = ed.recentCol().NewWindow()
			ed.errWin.SetFilename("+Errors")
		}
		q := ed.errWin.body.buf.End()
		ed.errWin.body.q0, ed.errWin.body.q1 = q, q
	}
	return ed.errWin.Write(b)
}

func (w *outputWriter) flush() {
	if w.ready {
		w.ed.errWin.flush()
		w.ready = false
	}
}
