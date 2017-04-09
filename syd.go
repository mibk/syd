package main

import (
	"log"
	"os"

	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/mouse"

	"github.com/mibk/syd/core"
	"github.com/mibk/syd/ui"
	"github.com/mibk/syd/ui/term"
	"github.com/mibk/syd/vi"
)

var (
	UI       = &term.UI{}
	filename = ""
)

func main() {
	log.SetPrefix("syd: ")
	log.SetFlags(0)
	if err := UI.Init(); err != nil {
		log.Fatalln("initializing ui:", err)
	}
	defer UI.Close()

	ed := &Editor{
		events: make(chan ui.Event),
		vi:     vi.NewParser(),
	}
	if len(os.Args) == 1 {
		ed.NewWindow()
	} else {
		for _, a := range os.Args[1:] {
			if err := ed.NewWindowFile(a); err != nil {
				panic(err)
			}
		}
	}
	ed.Main()
}

const (
	ModeNormal = iota
	ModeInsert
)

type Editor struct {
	events     chan ui.Event
	vi         *vi.Parser
	shouldQuit bool

	wins []*core.Window
	mode int
}

func (ed *Editor) Main() {
	for !ed.shouldQuit {
		for _, win := range ed.wins {
			win.LoadText()
		}
		UI.Flush()
		ev := <-ui.Events
		if ev == ui.Quit {
			return
		}
		switch ev := ev.(type) {
		case key.Event:
			UI.Push_Key_Event(ev)
		case mouse.Event:
			// Temporary reasons...
			UI.Push_Mouse_Event(ev)
		}
	}
}

func (ed *Editor) NewWindow() {
	ed.newWindow(core.BytesContent([]byte{}))
}

func (ed *Editor) NewWindowFile(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	mm, err := core.Mmap(f)
	if err != nil {
		return err
	}
	win := ed.newWindow(mm)
	win.SetFilename(filename)
	return nil
}

func (ed *Editor) newWindow(con core.Content) *core.Window {
	win := core.NewWindow(UI.NewWindow(), con)
	ed.wins = append(ed.wins, win)
	return win
}

func (ed *Editor) Close() error {
	for _, win := range ed.wins {
		// TODO: Check errors.
		win.Close()
	}
	return nil
}
