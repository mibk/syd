package core

import (
	"os"

	"github.com/mibk/syd/ui"
	"github.com/mibk/syd/ui/term"
	"github.com/mibk/syd/vi"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/mouse"
)

type Editor struct {
	ui         *term.UI
	events     chan ui.Event
	vi         *vi.Parser
	shouldQuit bool

	wins []*Window
	mode int
}

func NewEditor(u *term.UI) *Editor {
	return &Editor{
		ui:     u,
		events: make(chan ui.Event),
	}
}

func (ed *Editor) Main() {
	for !ed.shouldQuit {
		for _, win := range ed.wins {
			win.LoadText()
		}
		ed.ui.Flush()
		ev := <-ui.Events
		if ev == ui.Quit {
			return
		}
		switch ev := ev.(type) {
		case key.Event:
			ed.ui.Push_Key_Event(ev)
		case mouse.Event:
			// Temporary reasons...
			ed.ui.Push_Mouse_Event(ev)
		}
	}
}

func (ed *Editor) NewWindow() {
	ed.newWindow(BytesContent([]byte{}))
}

func (ed *Editor) NewWindowFile(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	mm, err := Mmap(f)
	if err != nil {
		return err
	}
	win := ed.newWindow(mm)
	win.SetFilename(filename)
	return nil
}

func (ed *Editor) newWindow(con Content) *Window {
	win := NewWindow(ed, ed.ui.NewWindow(), con)
	ed.wins = append(ed.wins, win)
	return win
}

func (ed *Editor) deleteWindow(todel *Window) {
	for i, win := range ed.wins {
		if win == todel {
			ed.wins = append(ed.wins[:i], ed.wins[i+1:]...)
			if len(ed.wins) == 0 {
				ed.shouldQuit = true
			}
			return
		}
	}
	panic("window not found")
}

func (ed *Editor) Close() error {
	for _, win := range ed.wins {
		// TODO: Check errors.
		win.Close()
	}
	return nil
}
