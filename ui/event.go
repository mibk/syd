package ui

var Events = make(chan Event)

type Event interface{}

var Quit = &struct{}{}

// TODO: Delete these constanst and use key.Event.Code.
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

// TODO: Delete once not needed by vi.
type KeyPress struct {
	Key       rune
	Ctrl, Alt bool
}
