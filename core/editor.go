package core

import (
	"os"

	"github.com/mibk/syd/pkg/undo"
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

func (ed *Editor) NewWindow() *Window {
	return ed.newWindow(BytesContent([]byte{}))
}

func (ed *Editor) NewWindowFile(filename string) (*Window, error) {
	f, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			win := ed.NewWindow()
			win.SetFilename(filename)
			return win, nil
		}
		return nil, err
	}
	mm, err := Mmap(f)
	if err != nil {
		return nil, err
	}
	win := ed.newWindow(mm)
	win.SetFilename(filename)
	return win, nil
}

func (ed *Editor) newWindow(con Content) *Window {
	window := ed.ui.NewWindow()
	buf := NewUndoBuffer(undo.NewBuffer(con.Bytes()))
	win := &Window{ed: ed, win: window, con: con, buf: buf}
	win.head = newText(win, &BasicBuffer{[]rune("\x00Exit New Del Put Undo Redo ")}, window.Head())
	win.body = newText(win, buf, window.Body())
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
