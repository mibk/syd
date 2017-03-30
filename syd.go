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
	UI       ui.Viewport = &term.UI{}
	filename             = ""
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
	setMappings(ed)
	ed.activeView.SetName(filename)
	go ed.RouteEvents()
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

func (ed *Editor) RouteEvents() {
	for ev := range ui.Events {
		if keyPress, ok := ev.(ui.KeyPress); ok && ed.mode == ModeNormal {
			ed.vi.Decode(keyPress)
			continue
		}
		ed.events <- ev
	}
}

func parseKeys(cmd string) []ui.KeyPress {
	events := make([]ui.KeyPress, len(cmd))
	for i, r := range []rune(cmd) {
		events[i] = ui.KeyPress{Key: r}
	}
	return events
}

func (ed *Editor) AddOperator(cmd []ui.KeyPress, fn func(*view.View, int)) {
	ed.vi.AddOperator(cmd, func(n int) { fn(ed.activeView, n) }, false)
}

func (ed *Editor) AddStringOperator(cmd string, fn func(*view.View, int)) {
	ed.AddOperator(parseKeys(cmd), fn)
}

func (ed *Editor) AddMotion(cmd []ui.KeyPress, fn func(*view.View, int)) {
	ed.vi.AddMotion(cmd, func(n int) { fn(ed.activeView, n) })
}

func (ed *Editor) AddStringMotion(cmd string, fn func(*view.View, int)) {
	ed.AddMotion(parseKeys(cmd), fn)
}

func (ed *Editor) Main() {
	for !ed.shouldQuit {
		ed.activeView.Render()
		select {
		case action := <-ed.vi.Actions:
			action()
		case ev := <-ed.events:
			if ev == ui.Quit {
				return
			}
			switch ev := ev.(type) {
			case ui.KeyPress:
				if ev.Key == ui.KeyEscape {
					ed.mode = ModeNormal
					continue
				}
				handleKeyPress(ed.activeView, ev)
			case mouse.Event:
				// Temporary reasons...
				UI.Push_Mouse_Event(ev)
			}
		}
	}
}
