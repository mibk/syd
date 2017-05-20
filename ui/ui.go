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

type Message int

const (
	_ Message = iota
	Delete
)

type Updater interface {
	Update(Message)
}

// The following interfaces are for refactoring purposes only.

type ResetRuneReader interface {
	// Reset resets the reader to the original offset.
	Reset()
	io.RuneReader
}

type UI interface {
	Tag() Text
	NewColumn(Model) Column
}

type Column interface {
	Updater
	NewWindow(Model) Window
	Tag() Text
}

type Window interface {
	Updater
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

type Model interface{}
