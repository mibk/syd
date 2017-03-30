package main

import (
	"log"
	"os"

	"golang.org/x/mobile/event/mouse"

	"github.com/edsrzf/mmap-go"
	"github.com/mibk/syd/core"
	"github.com/mibk/syd/pkg/undo"
	"github.com/mibk/syd/ui"
	"github.com/mibk/syd/ui/term"
	"github.com/mibk/syd/vi"
	"github.com/mibk/syd/view"
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

	var b []byte
	if len(os.Args) > 1 {
		filename = os.Args[1]
		m, err := readFile(filename)
		if err != nil {
			panic(err)
		}
		defer m.Unmap()
		b = []byte(m)
	}
	buf := undo.NewBuffer(b)

	win := UI.NewWindow()
	ed := &Editor{
		events:     make(chan ui.Event),
		vi:         vi.NewParser(),
		activeView: view.New(win, core.NewBuffer(buf)),
	}
	ed.activeView.SetFilename(filename)
	ed.Main()
}

func readFile(filename string) (mmap.MMap, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	m, err := mmap.Map(f, 0, 0)
	if err != nil {
		return nil, err
	}
	return m, nil
}

const (
	ModeNormal = iota
	ModeInsert
)

type Editor struct {
	events     chan ui.Event
	vi         *vi.Parser
	shouldQuit bool

	activeView *view.View
	mode       int
}

func (ed *Editor) Main() {
	for !ed.shouldQuit {
		ed.activeView.Render()
		ev := <-ui.Events
		if ev == ui.Quit {
			return
		}
		switch ev := ev.(type) {
		case ui.KeyPress:
			handleKeyPress(ed.activeView, ev)
		case mouse.Event:
			// Temporary reasons...
			UI.Push_Mouse_Event(ev)
		}
	}
}
