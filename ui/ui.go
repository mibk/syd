package ui

import (
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/mouse"
)

type MouseEventHandler func(p int, ev mouse.Event)

type KeyEventHandler func(ev key.Event)

const (
	ColQ0 = -1
	ColQ1 = -2
)
