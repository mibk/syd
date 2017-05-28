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
	NewColumn(Model) Column
}

type Column interface {
	Updater
	NewWindow(Model) Updater
}

type Model interface{}
