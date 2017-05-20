package ui

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
	Init(Model)
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
