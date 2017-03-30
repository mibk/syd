package ui

var Events = make(chan Event)

type Event interface{}

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

type KeyPress struct {
	Key       rune
	Ctrl, Alt bool
}
