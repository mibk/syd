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

	var con core.Content = core.BytesContent([]byte{})
	if len(os.Args) > 1 {
		filename = os.Args[1]
		f, err := os.Open(filename)
		if err != nil {
			panic(err)
		}
		con, err = core.Mmap(f)
		if err != nil {
			panic(err)
		}
	}

	win := UI.NewWindow()
	ed := &Editor{
		events:    make(chan ui.Event),
		vi:        vi.NewParser(),
		activeWin: core.NewWindow(win, con),
	}
	ed.activeWin.SetFilename(filename)
	defer ed.activeWin.Close()
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

	activeWin *core.Window
	mode      int
}

func (ed *Editor) Main() {
	for !ed.shouldQuit {
		ed.activeWin.Render()
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
