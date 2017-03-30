package term

import (
	"github.com/gdamore/tcell"
	"github.com/mibk/syd/ui"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/mouse"
)

func (t *UI) translateEvents() {
	for {
		termEv := t.screen.PollEvent()
		switch termEv := termEv.(type) {
		case *tcell.EventKey:
			ev := key.Event{Direction: key.DirPress}
			switch termEv.Key() {
			case tcell.KeyCtrlSpace:
				ev.Rune, ev.Modifiers = ' ', key.ModControl
			case tcell.KeyCtrlA:
				ev.Rune, ev.Modifiers = 'a', key.ModControl
			case tcell.KeyCtrlB:
				ev.Rune, ev.Modifiers = 'b', key.ModControl
			case tcell.KeyCtrlC:
				ev.Rune, ev.Modifiers = 'c', key.ModControl
			case tcell.KeyCtrlD:
				ev.Rune, ev.Modifiers = 'd', key.ModControl
			case tcell.KeyCtrlE:
				ev.Rune, ev.Modifiers = 'e', key.ModControl
			case tcell.KeyCtrlF:
				ev.Rune, ev.Modifiers = 'f', key.ModControl
			case tcell.KeyCtrlG:
				ev.Rune, ev.Modifiers = 'g', key.ModControl

			case tcell.KeyCtrlJ:
				ev.Rune, ev.Modifiers = 'j', key.ModControl
			case tcell.KeyCtrlK:
				ev.Rune, ev.Modifiers = 'k', key.ModControl
			case tcell.KeyCtrlL:
				ev.Rune, ev.Modifiers = 'l', key.ModControl

			case tcell.KeyCtrlN:
				ev.Rune, ev.Modifiers = 'n', key.ModControl
			case tcell.KeyCtrlO:
				ev.Rune, ev.Modifiers = 'o', key.ModControl
			case tcell.KeyCtrlP:
				ev.Rune, ev.Modifiers = 'p', key.ModControl
			case tcell.KeyCtrlQ:
				ev.Rune, ev.Modifiers = 'q', key.ModControl
			case tcell.KeyCtrlR:
				ev.Rune, ev.Modifiers = 'r', key.ModControl
			case tcell.KeyCtrlS:
				ev.Rune, ev.Modifiers = 's', key.ModControl
			case tcell.KeyCtrlT:
				ev.Rune, ev.Modifiers = 't', key.ModControl
			case tcell.KeyCtrlU:
				ev.Rune, ev.Modifiers = 'u', key.ModControl
			case tcell.KeyCtrlV:
				ev.Rune, ev.Modifiers = 'v', key.ModControl
			case tcell.KeyCtrlW:
				ev.Rune, ev.Modifiers = 'w', key.ModControl
			case tcell.KeyCtrlX:
				ev.Rune, ev.Modifiers = 'x', key.ModControl
			case tcell.KeyCtrlY:
				ev.Rune, ev.Modifiers = 'y', key.ModControl
			case tcell.KeyCtrlZ:
				ev.Rune, ev.Modifiers = 'z', key.ModControl

			case tcell.KeyEnter:
				ev.Rune = ui.KeyEnter
			case tcell.KeyTab:
				ev.Rune = '\t'
			case tcell.KeyBackspace, tcell.KeyBackspace2:
				ev.Rune = ui.KeyBackspace
			case tcell.KeyDelete:
				ev.Rune = ui.KeyDelete
			case tcell.KeyEscape:
				ev.Rune = ui.KeyEscape
			case tcell.KeyLeft:
				ev.Rune = ui.KeyLeft
			case tcell.KeyRight:
				ev.Rune = ui.KeyRight
			case tcell.KeyUp:
				ev.Rune = ui.KeyUp
			case tcell.KeyDown:
				ev.Rune = ui.KeyDown
			case tcell.KeyPgUp:
				ev.Rune = ui.KeyPageUp
			case tcell.KeyPgDn:
				ev.Rune = ui.KeyPageDown
			case tcell.KeyRune:
				ev.Rune = termEv.Rune()
			default:
				continue
			}

			mod := termEv.Modifiers()
			if mod&tcell.ModCtrl > 0 {
				ev.Modifiers |= key.ModControl
			}
			if mod&tcell.ModAlt > 0 {
				ev.Modifiers |= key.ModAlt
			}
			ui.Events <- ev
		case *tcell.EventMouse:
			x, y := termEv.Position()
			ev := mouse.Event{
				X: float32(x),
				Y: float32(y),
			}
			btns := termEv.Buttons()
			switch {
			case btns == 0:
				if t.wasBtnPressed {
					// TODO: Send which button was released.
					t.wasBtnPressed = false
					ev.Direction = mouse.DirRelease
				}
				fallthrough
			case t.wasBtnPressed:
				ui.Events <- ev
				continue
			}
			ev.Direction = mouse.DirPress
			switch {
			case btns&tcell.Button1 > 0:
				ev.Button = mouse.ButtonLeft
				t.wasBtnPressed = true
			case btns&tcell.Button2 > 0:
				ev.Button = mouse.ButtonMiddle
				t.wasBtnPressed = true
			case btns&tcell.Button3 > 0:
				ev.Button = mouse.ButtonRight
				t.wasBtnPressed = true
			case btns&tcell.WheelUp > 0:
				ev.Button = mouse.ButtonWheelUp
				ev.Direction = mouse.DirStep
			case btns&tcell.WheelDown > 0:
				ev.Button = mouse.ButtonWheelDown
				ev.Direction = mouse.DirStep
			default:
				continue
			}
			ui.Events <- ev
		}
	}
}
