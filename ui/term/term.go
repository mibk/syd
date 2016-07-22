package term

import (
	"github.com/mibk/syd/ui"
	"github.com/nsf/termbox-go"
)

const (
	AttrDefault = 0
	AttrReverse = 1 << iota
	AttrBold
)

type UI struct{}

func (ui UI) Init() {
	termbox.Init()
	go ui.translateEvents()
}

func (UI) Close() {
	termbox.Close()
}

func (UI) SetCursor(x, y int) {
	termbox.SetCursor(x, y)
}

func (UI) SetCell(x, y int, r rune, attrs uint8) {
	a := termbox.ColorDefault
	if attrs&AttrReverse == AttrReverse {
		a |= termbox.AttrReverse
	}
	if attrs&AttrBold == AttrBold {
		a |= termbox.AttrBold
	}
	termbox.SetCell(x, y, r, a, a)
}

func (UI) Clear() {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
}

func (UI) Flush() {
	termbox.Flush()
}

func (UI) Size() (w, h int) {
	return termbox.Size()
}

func (UI) translateEvents() {
	for {
		termEv := termbox.PollEvent()
		switch termEv.Type {
		case termbox.EventKey:
			if termEv.Ch != 0 {
				ui.Events <- ui.KeyPress{Key: termEv.Ch}
				continue
			}
			var ev ui.KeyPress
			switch termEv.Key {
			case termbox.KeyCtrlSpace:
				ev = ui.KeyPress{Key: ' ', Ctrl: true}
			case termbox.KeyCtrlA:
				ev = ui.KeyPress{Key: 'a', Ctrl: true}
			case termbox.KeyCtrlB:
				ev = ui.KeyPress{Key: 'b', Ctrl: true}
			case termbox.KeyCtrlC:
				ev = ui.KeyPress{Key: 'c', Ctrl: true}
			case termbox.KeyCtrlD:
				ev = ui.KeyPress{Key: 'd', Ctrl: true}
			case termbox.KeyCtrlE:
				ev = ui.KeyPress{Key: 'e', Ctrl: true}
			case termbox.KeyCtrlF:
				ev = ui.KeyPress{Key: 'f', Ctrl: true}
			case termbox.KeyCtrlG:
				ev = ui.KeyPress{Key: 'g', Ctrl: true}
			case termbox.KeyCtrlH:
				ev = ui.KeyPress{Key: 'h', Ctrl: true}
			// Ctrl+I is the same as termbox.KeyTab
			case termbox.KeyCtrlJ:
				ev = ui.KeyPress{Key: 'j', Ctrl: true}
			case termbox.KeyCtrlK:
				ev = ui.KeyPress{Key: 'k', Ctrl: true}
			case termbox.KeyCtrlL:
				ev = ui.KeyPress{Key: 'l', Ctrl: true}
			// Ctrl+M is the same as termbox.KeyEnter
			case termbox.KeyCtrlN:
				ev = ui.KeyPress{Key: 'n', Ctrl: true}
			case termbox.KeyCtrlO:
				ev = ui.KeyPress{Key: 'o', Ctrl: true}
			case termbox.KeyCtrlP:
				ev = ui.KeyPress{Key: 'p', Ctrl: true}
			case termbox.KeyCtrlQ:
				ev = ui.KeyPress{Key: 'q', Ctrl: true}
			case termbox.KeyCtrlR:
				ev = ui.KeyPress{Key: 'r', Ctrl: true}
			case termbox.KeyCtrlS:
				ev = ui.KeyPress{Key: 's', Ctrl: true}
			case termbox.KeyCtrlT:
				ev = ui.KeyPress{Key: 't', Ctrl: true}
			case termbox.KeyCtrlU:
				ev = ui.KeyPress{Key: 'u', Ctrl: true}
			case termbox.KeyCtrlV:
				ev = ui.KeyPress{Key: 'v', Ctrl: true}
			case termbox.KeyCtrlW:
				ev = ui.KeyPress{Key: 'w', Ctrl: true}
			case termbox.KeyCtrlX:
				ev = ui.KeyPress{Key: 'x', Ctrl: true}
			case termbox.KeyCtrlY:
				ev = ui.KeyPress{Key: 'y', Ctrl: true}
			case termbox.KeyCtrlZ:
				ev = ui.KeyPress{Key: 'z', Ctrl: true}

			case termbox.KeySpace:
				ev = ui.KeyPress{Key: ' '}
			case termbox.KeyTab:
				ev = ui.KeyPress{Key: '\t'}
			case termbox.KeyEnter:
				ev = ui.KeyPress{Key: ui.KeyEnter}
			case termbox.KeyBackspace2:
				ev = ui.KeyPress{Key: ui.KeyBackspace}
			case termbox.KeyDelete:
				ev = ui.KeyPress{Key: ui.KeyDelete}
			case termbox.KeyEsc:
				ev = ui.KeyPress{Key: ui.KeyEscape}

			case termbox.KeyArrowLeft:
				ev = ui.KeyPress{Key: ui.KeyLeft}
			case termbox.KeyArrowRight:
				ev = ui.KeyPress{Key: ui.KeyRight}
			case termbox.KeyArrowUp:
				ev = ui.KeyPress{Key: ui.KeyUp}
			case termbox.KeyArrowDown:
				ev = ui.KeyPress{Key: ui.KeyDown}

			case termbox.KeyPgup:
				ev = ui.KeyPress{Key: ui.KeyPageUp}
			case termbox.KeyPgdn:
				ev = ui.KeyPress{Key: ui.KeyPageDown}

			default:
				continue
			}
			ui.Events <- ev
		}
	}
}
