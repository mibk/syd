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

const (
	MouseButton1 = 1 + iota
	MouseButton2
	MouseButton3

	MouseWheelUp
	MouseWheelDown
)

type MouseBtnPress struct {
	X, Y   int
	Button int
}

type MouseBtnRelease struct {
	X, Y int
}

type MouseMove struct {
	X, Y int
}
