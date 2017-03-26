package main

import (
	"bytes"
	"log"
	"os"
	"os/exec"
	"strings"
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
	var (
		lastQ     int64 = -1
		timestamp time.Time
	)
	for !ed.shouldQuit {
		ed.activeView.Render()
		select {
		case action := <-ed.vi.Actions:
			action()
		case ev := <-ed.events:
			switch ev := ev.(type) {
			case ui.KeyPress:
				if ev.Key == ui.KeyEscape {
					ed.mode = ModeNormal
					continue
				}
				handleKeyPress(ed.activeView, ev)
			case ui.MouseBtnPress:
				switch ev.Button {
				case ui.MouseButton1:
					x, y := ed.activeView.Position()
					p := ed.activeView.Frame().CharsUntilXY(ev.X-x, ev.Y-y-ui.HeadHeight)
					q := ed.activeView.Origin() + int64(p)
					if time.Since(timestamp) < 300*time.Millisecond {
						ed.activeView.Select(dblclick(ed.activeView, q))
						ed.activeView.Frame().SetWantCol(ui.ColQ0)
						lastQ = -1
						continue
					}
					ed.activeView.Select(q, q)
					ed.activeView.Frame().SetWantCol(ui.ColQ0)
					lastQ = q
					timestamp = time.Now()
				case ui.MouseButton2:
					// This is just ugly proof of concept.
					x, y := ed.activeView.Position()
					p := ed.activeView.Frame().CharsUntilXY(ev.X-x, ev.Y-y-ui.HeadHeight)
					q := ed.activeView.Origin() + int64(p)
					q0, q1 := dblclick(ed.activeView, q)
					var cmd []rune
					for i := q0; i < q1; i++ {
						cmd = append(cmd, ed.activeView.ReadRuneAt(i))
					}
					ed.Execute(string(cmd))
				case ui.MouseWheelUp:
					scrollUp(ed.activeView, 3)
				case ui.MouseWheelDown:
					scrollDown(ed.activeView, 3)
				}
			case ui.MouseBtnRelease:
				lastQ = -1
			case ui.MouseMove:
				if lastQ < 0 {
					continue
				}
				x, y := ed.activeView.Position()
				p := ed.activeView.Frame().CharsUntilXY(ev.X-x, ev.Y-y-ui.HeadHeight)
				q0, q1 := lastQ, ed.activeView.Origin()+int64(p)
				if q1 < q0 {
					q0, q1 = q1, q0
				}
				ed.activeView.Select(q0, q1)
			}
		}
	}
}

func (ed *Editor) Execute(command string) {
	switch command {
	case "Exit":
		ed.shouldQuit = true
	case "Put":
		if filename != "" {
			if err := saveFile(filename, ed.activeView); err != nil {
				panic(err)
			}
		}
	case "Undo":
		ed.activeView.Undo()
	case "Redo":
		ed.activeView.Redo()
	default:
		v := ed.activeView
		var selected []rune
		q0, q1 := v.Selected()
		for p := q0; p < q1; p++ {
			r := v.ReadRuneAt(p)
			selected = append(selected, r)
		}
		var buf bytes.Buffer
		rd := strings.NewReader(string(selected))
		cmd := exec.Command(command)
		cmd.Stdin = rd
		cmd.Stdout = &buf
		// TODO: Redirect stderr somewhere.
		if err := cmd.Run(); err != nil {
			panic(err)
		}
		s := buf.String()
		v.Insert(s)
		v.Select(q0, q0+int64(utf8.RuneCountInString(s)))
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
