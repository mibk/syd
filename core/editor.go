package core

import "github.com/mibk/syd/ui"

type Editor struct {
	ui ui.UI

	tag *Text

	// TODO: Make it just an io.Reader so it's easier to work
	// with.
	errWin *Window

	firstCol *Column
	wins     map[string]*Window
	mode     int
}

func NewEditor() *Editor {
	return &Editor{
		wins: make(map[string]*Window),
	}
}

func (ed *Editor) SetUI(u ui.UI) {
	ed.ui = u
	ed.tag = newText(ed, &BasicBuffer{[]rune("Newcol Exit ")}, u.Tag())
	q := ed.tag.buf.End()
	ed.tag.q0, ed.tag.q1 = q, q
}

func (ed *Editor) NewColumn() *Column {
	col := &Column{ed: ed}

	sentinel := &Column{next: ed.firstCol}
	prev := sentinel
	for prev.next != nil {
		prev = prev.next
	}
	prev.next = col
	ed.firstCol = sentinel.next

	column := ed.ui.NewColumn(col)
	col.col = column
	col.tag = newText(col, &BasicBuffer{[]rune("New Delcol ")}, col.col.Tag())
	q := col.tag.buf.End()
	col.tag.q0, col.tag.q1 = q, q
	return col
}

func (ed *Editor) MoveColumn(col *Column, x float64) {
	target := ed.firstCol
	for target != nil {
		if x < target.right() {
			break
		}
		target = target.next
	}
	if target == nil {
		return
	}

	if x == target.x {
		// TODO: Adjust position. See the method Column.MoveWindow.
		return
	}

	if col == target || (target.next != nil && col == target.next) {
		if col == ed.firstCol {
			return
		}
	} else {
		ed.removeColumn(col)
		col.next = target.next
		target.next = col
	}
	col.SetX(x)
}

func (ed *Editor) Close() error {
	col := ed.firstCol
	for col != nil {
		col.Close()
		col = col.next
	}
	return nil
}

func (ed *Editor) recentCol() *Column {
	if ed.firstCol == nil {
		return ed.NewColumn()
	}
	return ed.firstCol
}

func (ed *Editor) removeColumn(todel *Column) {
	sentinel := &Column{next: ed.firstCol}
	col := sentinel
	for col.next != nil {
		if col.next == todel {
			col.next = todel.next
			ed.firstCol = sentinel.next
			if ed.firstCol != nil {
				ed.firstCol.SetX(0)
			}
			return
		}
		col = col.next
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
