package ui

const (
	KeyEnter     = '\n'
	KeyEscape    = 0x1B
	KeyBackspace = 0x08
	KeyDelete    = 0x7F

	KeyUp = 0xFFFF - iota
	KeyDown
	KeyLeft
	KeyRight

	KeyPageUp
	KeyPageDown
)

type Event interface{}

type KeyPress struct {
	Key       rune
	Ctrl, Alt bool
}

var Events = make(chan Event)
