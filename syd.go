package main

import (
	"log"
	"os"
	"time"

	"github.com/edsrzf/mmap-go"
	"github.com/mibk/syd/core"
	"github.com/mibk/syd/ui"
	"github.com/mibk/syd/ui/term"
	"github.com/mibk/syd/undo"
	"github.com/mibk/syd/vi"
	"github.com/mibk/syd/view"
)

var (
	win      term.UI
	filename = ""
)

func main() {
	log.SetPrefix("syd: ")
	log.SetFlags(0)
	if err := win.Init(); err != nil {
		log.Fatalln("initializing ui:", err)
	}
	defer win.Close()

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

	syd := &Syd{
		events:     make(chan ui.Event),
		vi:         vi.NewParser(),
		buffer:     buf,
		activeView: view.New(core.NewBuffer(buf)),
	}
	mapCommands(syd)
	go syd.RouteEvents()
	syd.NormalMode()
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

func print(x, y int, s string, attrs uint8) {
	for _, r := range []rune(s) {
		win.SetCell(x, y, r, attrs)
		x++
	}
}

const (
	ModeNormal = iota
	ModeInsert
)

type Syd struct {
	events     chan ui.Event
	vi         *vi.Parser
	shouldQuit bool

	buffer     *undo.Buffer // TODO: remove
	activeView *view.View
	mode       int
}

func (syd *Syd) RouteEvents() {
	for ev := range ui.Events {
		if keyPress, ok := ev.(ui.KeyPress); ok && syd.mode == ModeNormal {
			syd.vi.Decode(keyPress)
			continue
		}
		syd.events <- ev
	}
}

func parseKeys(cmd string) []ui.KeyPress {
	events := make([]ui.KeyPress, len(cmd))
	for i, r := range []rune(cmd) {
		events[i] = ui.KeyPress{Key: r}
	}
	return events
}

func (syd *Syd) AddCommand(cmd []ui.KeyPress, fn func(*view.View, int)) {
	syd.vi.AddCommand(cmd, func(n int) { fn(syd.activeView, n) })
}

func (syd *Syd) AddStringCommand(cmd string, fn func(*view.View, int)) {
	syd.AddCommand(parseKeys(cmd), fn)
}

func (syd *Syd) AddMotion(cmd []ui.KeyPress, fn func(*view.View, int)) {
	syd.vi.AddMotion(cmd, func(n int) { fn(syd.activeView, n) })
}

func (syd *Syd) AddStringMotion(cmd string, fn func(*view.View, int)) {
	syd.AddMotion(parseKeys(cmd), fn)
}

func (syd *Syd) NormalMode() {
	for !syd.shouldQuit {
		w, h := win.Size()
		syd.activeView.SetSize(w, h-2) // 2 for the footer
		syd.activeView.Render(win)
		syd.printFoot()
		win.Flush()

		action := <-syd.vi.Actions
		action()
	}
}

func (syd *Syd) InsertMode() {
	syd.mode = ModeInsert
	defer func() { syd.mode = ModeNormal }()
	for {
		w, h := win.Size()
		syd.activeView.SetSize(w, h-2) // 2 for the footer
		syd.activeView.Render(win)
		syd.printFoot()
		print(0, h-1, "-- INSERT --", term.AttrBold)
		win.Flush()
		select {
		case ev := <-syd.events:
			switch ev := ev.(type) {
			case ui.KeyPress:
				if ev.Key == ui.KeyEscape {
					return
				}
				handleKeyPress(syd.activeView, ev)
			}
		case <-time.After(3 * time.Second):
			syd.buffer.CommitChanges()
		}
	}
}

func (syd *Syd) printFoot() {
	w, h := win.Size()
	for x := 0; x < w; x++ {
		win.SetCell(x, h-2, ' ', term.AttrReverse|term.AttrBold)
	}
	filename := filename
	if filename == "" {
		filename = "[No Name]"
	}
	if syd.buffer.Modified() {
		filename += " [+]"
	}
	print(0, h-2, filename, term.AttrReverse|term.AttrBold)
}
