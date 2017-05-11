package ui

import (
	"io"

	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/mouse"
)

type MouseEventHandler func(p int, ev mouse.Event)

type KeyEventHandler func(ev key.Event)

const (
	ColQ0 = -1
	ColQ1 = -2
)

// The following interfaces are for refactoring purposes only.

type WindowMovedHandler func(win Window, from Column)

type ResetRuneReader interface {
	// Reset resets the reader to the original offset.
	Reset()
	io.RuneReader
}

type UI interface {
	Tag() Text
	Flush()
	Push_Key_Event(key.Event)
	Push_Mouse_Event(mouse.Event)
	NewColumn() Column
}

type Column interface {
	NewWindow() Window
	Delete()
	Tag() Text
	OnWindowMoved(WindowMovedHandler)
}

type Window interface {
	SetDirty(bool)
	Delete()
	Tag() Text
	Body() Text
}

type Text interface {
	Init(ResetRuneReader)
	Size() (w, h int)
	OnMouseEvent(MouseEventHandler)
	OnKeyEvent(KeyEventHandler)
	Select(p0, p1 int)
	Reload() error
	Frame() Frame
}

type Frame interface {
	Nchars() int
	SelectionLines() (int, int)
	CharsUntilXY(x, y int) int
	Lines() int
	WantCol() int
	SetWantCol(int)
}
