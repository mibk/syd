package main

import (
	"log"
	"os"
	"time"
	"unicode/utf8"

	"github.com/edsrzf/mmap-go"
	"github.com/mibk/syd/core"
	"github.com/mibk/syd/pkg/undo"
	"github.com/mibk/syd/ui"
	"github.com/mibk/syd/ui/term"
	"github.com/mibk/syd/vi"
	"github.com/mibk/syd/view"
)

var (
	win      = &term.UI{}
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
		activeView: view.New(win, core.NewBuffer(buf)),
	}
	setMappings(syd)
	go syd.RouteEvents()
	syd.Main()
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

func (syd *Syd) AddOperator(cmd []ui.KeyPress, fn func(*view.View, int)) {
	syd.vi.AddOperator(cmd, func(n int) { fn(syd.activeView, n) }, false)
}

func (syd *Syd) AddStringOperator(cmd string, fn func(*view.View, int)) {
	syd.AddOperator(parseKeys(cmd), fn)
}

func (syd *Syd) AddMotion(cmd []ui.KeyPress, fn func(*view.View, int)) {
	syd.vi.AddMotion(cmd, func(n int) { fn(syd.activeView, n) })
}

func (syd *Syd) AddStringMotion(cmd string, fn func(*view.View, int)) {
	syd.AddMotion(parseKeys(cmd), fn)
}

func (syd *Syd) Main() {
	var (
		lastQ     int64 = -1
		timestamp time.Time
	)
	for !syd.shouldQuit {
		syd.activeView.Render()
		select {
		case action := <-syd.vi.Actions:
			action()
		case ev := <-syd.events:
			switch ev := ev.(type) {
			case ui.KeyPress:
				if ev.Key == ui.KeyEscape {
					syd.mode = ModeNormal
					continue
				}
				handleKeyPress(syd.activeView, ev)
			case ui.MouseBtnPress:
				switch ev.Button {
				case ui.MouseButton1:
					p := syd.activeView.Frame().CharsUntilXY(ev.X, ev.Y)
					q := syd.activeView.Origin() + int64(p)
					if time.Since(timestamp) < 300*time.Millisecond {
						syd.activeView.Select(dblclick(syd.activeView, q))
						syd.activeView.Frame().SetWantCol(ui.ColQ0)
						lastQ = -1
						continue
					}
					syd.activeView.Select(q, q)
					syd.activeView.Frame().SetWantCol(ui.ColQ0)
					lastQ = q
					timestamp = time.Now()
				case ui.MouseButton2:
					// This is just ugly proof of concept.
					p := syd.activeView.Frame().CharsUntilXY(ev.X, ev.Y)
					q := syd.activeView.Origin() + int64(p)
					q0, q1 := dblclick(syd.activeView, q)
					var cmd []rune
					for i := q0; i < q1; i++ {
						cmd = append(cmd, syd.activeView.ReadRuneAt(i))
					}
					syd.Execute(string(cmd))
				case ui.MouseWheelUp:
					scrollUp(syd.activeView, 3)
				case ui.MouseWheelDown:
					scrollDown(syd.activeView, 3)
				}
			case ui.MouseBtnRelease:
				lastQ = -1
			case ui.MouseMove:
				if lastQ < 0 {
					continue
				}
				p := syd.activeView.Frame().CharsUntilXY(ev.X, ev.Y)
				q0, q1 := lastQ, syd.activeView.Origin()+int64(p)
				if q1 < q0 {
					q0, q1 = q1, q0
				}
				syd.activeView.Select(q0, q1)
			}
		}
	}
}

func (syd *Syd) Execute(cmd string) {
	switch cmd {
	case "Put":
		if filename != "" {
			if err := saveFile(filename, syd.activeView); err != nil {
				panic(err)
			}
		}
	}
}

func saveFile(filename string, v *view.View) error {
	// TODO: Read bytes directly from the undo.Buffer.
	f, err := os.Create(filename + "~")
	if err != nil {
		return err
	}

	var buf [64]byte
	var i int

	for p := int64(0); ; p++ {
		r := v.ReadRuneAt(p)
		if r == view.EOF || len(buf[i:]) < utf8.UTFMax {
			if _, err := f.Write(buf[:i]); err != nil {
				return err
			}
			i = 0
		}
		if r == view.EOF {
			break
		}
		i += utf8.EncodeRune(buf[i:], r)
	}
	f.Close()

	return os.Rename(filename+"~", filename)
}
