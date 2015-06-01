package console

import (
	"github.com/mibk/syd/event"

	"github.com/mibk/syd/third_party/github.com/nsf/termbox-go"
)

const (
	AttrDefault = 0
	AttrReverse = 1 << iota
	AttrBold
)

type Console struct{}

func (c Console) Init() {
	termbox.Init()
	go c.translateEvents()
}

func (Console) Close() {
	termbox.Close()
}

func (Console) SetCursor(x, y int) {
	termbox.SetCursor(x, y)
}

func (Console) SetCell(x, y int, r rune, attrs uint8) {
	a := termbox.ColorDefault
	if attrs&AttrReverse == AttrReverse {
		a |= termbox.AttrReverse
	}
	if attrs&AttrBold == AttrBold {
		a |= termbox.AttrBold
	}
	termbox.SetCell(x, y, r, a, a)
}

func (Console) Clear() {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
}

func (Console) Flush() {
	termbox.Flush()
}

func (Console) Size() (w, h int) {
	return termbox.Size()
}

func (Console) translateEvents() {
Loop:
	for {
		ev := termbox.PollEvent()
		switch ev.Type {
		case termbox.EventKey:
			var outEv event.KeyPress
			if ev.Ch == 0 {
				switch ev.Key {
				case termbox.KeyCtrlSpace:
					outEv = event.KeyPress{Key: ' ', Ctrl: true}
				case termbox.KeyCtrlA:
					outEv = event.KeyPress{Key: 'a', Ctrl: true}
				case termbox.KeyCtrlB:
					outEv = event.KeyPress{Key: 'b', Ctrl: true}
				case termbox.KeyCtrlC:
					outEv = event.KeyPress{Key: 'c', Ctrl: true}
				case termbox.KeyCtrlD:
					outEv = event.KeyPress{Key: 'd', Ctrl: true}
				case termbox.KeyCtrlE:
					outEv = event.KeyPress{Key: 'e', Ctrl: true}
				case termbox.KeyCtrlF:
					outEv = event.KeyPress{Key: 'f', Ctrl: true}
				case termbox.KeyCtrlG:
					outEv = event.KeyPress{Key: 'g', Ctrl: true}
				case termbox.KeyCtrlH:
					outEv = event.KeyPress{Key: 'h', Ctrl: true}
				// Ctrl+I is the same as termbox.KeyTab
				case termbox.KeyCtrlJ:
					outEv = event.KeyPress{Key: 'j', Ctrl: true}
				case termbox.KeyCtrlK:
					outEv = event.KeyPress{Key: 'k', Ctrl: true}
				case termbox.KeyCtrlL:
					outEv = event.KeyPress{Key: 'l', Ctrl: true}
				// Ctrl+M is the same as termbox.KeyEnter
				case termbox.KeyCtrlN:
					outEv = event.KeyPress{Key: 'n', Ctrl: true}
				case termbox.KeyCtrlO:
					outEv = event.KeyPress{Key: 'o', Ctrl: true}
				case termbox.KeyCtrlP:
					outEv = event.KeyPress{Key: 'p', Ctrl: true}
				case termbox.KeyCtrlQ:
					outEv = event.KeyPress{Key: 'q', Ctrl: true}
				case termbox.KeyCtrlR:
					outEv = event.KeyPress{Key: 'r', Ctrl: true}
				case termbox.KeyCtrlS:
					outEv = event.KeyPress{Key: 's', Ctrl: true}
				case termbox.KeyCtrlT:
					outEv = event.KeyPress{Key: 't', Ctrl: true}
				case termbox.KeyCtrlU:
					outEv = event.KeyPress{Key: 'u', Ctrl: true}
				case termbox.KeyCtrlV:
					outEv = event.KeyPress{Key: 'v', Ctrl: true}
				case termbox.KeyCtrlW:
					outEv = event.KeyPress{Key: 'w', Ctrl: true}
				case termbox.KeyCtrlX:
					outEv = event.KeyPress{Key: 'x', Ctrl: true}
				case termbox.KeyCtrlY:
					outEv = event.KeyPress{Key: 'y', Ctrl: true}
				case termbox.KeyCtrlZ:
					outEv = event.KeyPress{Key: 'z', Ctrl: true}

				case termbox.KeySpace:
					outEv = event.KeyPress{Key: ' '}
				case termbox.KeyTab:
					outEv = event.KeyPress{Key: '\t'}
				case termbox.KeyEnter:
					outEv = event.KeyPress{Key: event.Enter}
				case termbox.KeyBackspace2:
					outEv = event.KeyPress{Key: event.Backspace}
				case termbox.KeyDelete:
					outEv = event.KeyPress{Key: event.Delete}
				case termbox.KeyEsc:
					outEv = event.KeyPress{Key: event.Escape}

				default:
					continue Loop
				}
			} else {
				outEv = event.KeyPress{Key: event.Key(ev.Ch)}
			}
			event.MakeEvent(outEv)
		}
	}
}
