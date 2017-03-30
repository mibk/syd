package ui

import "golang.org/x/mobile/event/mouse"

// TODO: This is for temporary reasons only. Remove!
const HeadHeight = 2

type MouseEventHandler func(p int, ev mouse.Event)

const (
	ColQ0 = -1
	ColQ1 = -2
)
