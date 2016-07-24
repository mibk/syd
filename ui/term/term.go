package term

import (
	"github.com/gdamore/tcell"
	"github.com/mibk/syd/ui"
)

const (
	AttrDefault = 0
	AttrReverse = 1 << iota
	AttrBold
)

type UI struct {
	screen tcell.Screen
}

func (t *UI) Init() error {
	sc, err := tcell.NewScreen()
	if err != nil {
		return err
	}
	if err := sc.Init(); err != nil {
		return err
	}
	t.screen = sc
	go t.translateEvents()
	return nil
}

func (t *UI) Close() error {
	t.screen.Fini()
	return nil
}

func (t *UI) SetCursor(x, y int) {
	t.screen.ShowCursor(x, y)
}

func (t *UI) SetCell(x, y int, r rune, attrs uint8) {
	st := tcell.StyleDefault
	if attrs&AttrReverse > 0 {
		st = st.Reverse(true)
	}
	if attrs&AttrBold > 0 {
		st = st.Bold(true)
	}
	t.screen.SetContent(x, y, r, nil, st)
}

func (t *UI) Clear() { t.screen.Clear() }

func (t *UI) Flush() { t.screen.Show() }

func (t *UI) Size() (w, h int) { return t.screen.Size() }

func (t *UI) translateEvents() {
	for {
		termEv := t.screen.PollEvent()
		switch termEv := termEv.(type) {
		case *tcell.EventKey:
			var ev ui.KeyPress
			switch termEv.Key() {
			case tcell.KeyCtrlSpace:
				ev.Key, ev.Ctrl = ' ', true
			case tcell.KeyCtrlA:
				ev.Key, ev.Ctrl = 'a', true
			case tcell.KeyCtrlB:
				ev.Key, ev.Ctrl = 'b', true
			case tcell.KeyCtrlC:
				ev.Key, ev.Ctrl = 'c', true
			case tcell.KeyCtrlD:
				ev.Key, ev.Ctrl = 'd', true
			case tcell.KeyCtrlE:
				ev.Key, ev.Ctrl = 'e', true
			case tcell.KeyCtrlF:
				ev.Key, ev.Ctrl = 'f', true
			case tcell.KeyCtrlG:
				ev.Key, ev.Ctrl = 'g', true

			case tcell.KeyCtrlJ:
				ev.Key, ev.Ctrl = 'j', true
			case tcell.KeyCtrlK:
				ev.Key, ev.Ctrl = 'k', true
			case tcell.KeyCtrlL:
				ev.Key, ev.Ctrl = 'l', true

			case tcell.KeyCtrlN:
				ev.Key, ev.Ctrl = 'n', true
			case tcell.KeyCtrlO:
				ev.Key, ev.Ctrl = 'o', true
			case tcell.KeyCtrlP:
				ev.Key, ev.Ctrl = 'p', true
			case tcell.KeyCtrlQ:
				ev.Key, ev.Ctrl = 'q', true
			case tcell.KeyCtrlR:
				ev.Key, ev.Ctrl = 'r', true
			case tcell.KeyCtrlS:
				ev.Key, ev.Ctrl = 's', true
			case tcell.KeyCtrlT:
				ev.Key, ev.Ctrl = 't', true
			case tcell.KeyCtrlU:
				ev.Key, ev.Ctrl = 'u', true
			case tcell.KeyCtrlV:
				ev.Key, ev.Ctrl = 'v', true
			case tcell.KeyCtrlW:
				ev.Key, ev.Ctrl = 'w', true
			case tcell.KeyCtrlX:
				ev.Key, ev.Ctrl = 'x', true
			case tcell.KeyCtrlY:
				ev.Key, ev.Ctrl = 'y', true
			case tcell.KeyCtrlZ:
				ev.Key, ev.Ctrl = 'z', true

			case tcell.KeyEnter:
				ev.Key = ui.KeyEnter
			case tcell.KeyTab:
				ev.Key = '\t'
			case tcell.KeyBackspace, tcell.KeyBackspace2:
				ev.Key = ui.KeyBackspace
			case tcell.KeyDelete:
				ev.Key = ui.KeyDelete
			case tcell.KeyEscape:
				ev.Key = ui.KeyEscape
			case tcell.KeyLeft:
				ev.Key = ui.KeyLeft
			case tcell.KeyRight:
				ev.Key = ui.KeyRight
			case tcell.KeyUp:
				ev.Key = ui.KeyUp
			case tcell.KeyDown:
				ev.Key = ui.KeyDown
			case tcell.KeyPgUp:
				ev.Key = ui.KeyPageUp
			case tcell.KeyPgDn:
				ev.Key = ui.KeyPageDown
			case tcell.KeyRune:
				ev.Key = termEv.Rune()
			default:
				continue
			}

			mod := termEv.Modifiers()
			if mod&tcell.ModCtrl > 0 {
				ev.Ctrl = true
			}
			if mod&tcell.ModAlt > 0 {
				ev.Alt = true
			}
			ui.Events <- ev
		}
	}
}
