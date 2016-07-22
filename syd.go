package main

import (
	"os"
	"time"

	"github.com/edsrzf/mmap-go"
	"github.com/mibk/syd/core"
	"github.com/mibk/syd/event"
	"github.com/mibk/syd/ui/term"
	"github.com/mibk/syd/undo"
	"github.com/mibk/syd/view"
)

var (
	ui       term.UI
	filename = ""

	buffer   *undo.Buffer
	viewport *view.View
)

func main() {
	ui.Init()
	defer ui.Close()

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
	buffer = undo.NewBuffer(b)
	viewport = view.New(core.NewBuffer(buffer))
	insertMode()
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

func insertMode() {
	for {
		w, h := ui.Size()
		viewport.SetSize(w, h-2) // 2 for the footer
		viewport.Render(ui)
		printFoot()
		print(0, h-1, "-- INSERT --", term.AttrBold)
		ui.Flush()
		select {
		case ev := <-event.Events:
			switch ev := ev.(type) {
			case event.KeyPress:
				if ev.Key == 'x' && ev.Ctrl {
					return
				}
				viewport.Type(ev)
			}
		case <-time.After(3 * time.Second):
			buffer.CommitChanges()
		}
	}
}

func print(x, y int, s string, attrs uint8) {
	for _, r := range []rune(s) {
		ui.SetCell(x, y, r, attrs)
		x++
	}
}

func printFoot() {
	w, h := ui.Size()
	for x := 0; x < w; x++ {
		ui.SetCell(x, h-2, ' ', term.AttrReverse|term.AttrBold)
	}
	filename := filename
	if filename == "" {
		filename = "[No Name]"
	}
	if buffer.Modified() {
		filename += " [+]"
	}
	print(0, h-2, filename, term.AttrReverse|term.AttrBold)
}
