package event

type Key rune

const (
	Enter     = '\n'
	Escape    = 0x001B
	Backspace = 0x0008
	Delete    = 0x007F

	Up    = 0x2191
	Down  = 0x2193
	Left  = 0x2190
	Right = 0x2192
)

type Event interface{}

type KeyPress struct {
	Key       Key
	Ctrl, Alt bool
}

var events = make(chan Event, 10)

func MakeEvent(ev Event) {
	events <- ev
}

func PollEvent() Event {
	ev := <-events
	return ev
}
