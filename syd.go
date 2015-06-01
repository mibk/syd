package main

import (
	"os"
	"unicode/utf8"

	"github.com/mibk/syd/event"
	"github.com/mibk/syd/text"
	"github.com/mibk/syd/ui/console"
	"github.com/mibk/syd/view"

	"github.com/mibk/syd/third_party/github.com/edsrzf/mmap-go"
)

var (
	ui       console.Console
	filename = "[No Name]"
)

func main() {
	ui.Init()
	defer ui.Close()

	var initContent []byte
	if len(os.Args) > 1 {
		filename = os.Args[1]
		m, err := readFile(filename)
		if err != nil {
			panic(err)
		}
		defer m.Unmap()
		initContent = []byte(m)
	} else {
		initContent = []byte("\n")
	}

	t := text.New(initContent)
	v := view.New(t.GetReader())
	normalMode(v, t)
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

func normalMode(v *view.View, t *text.Text) {
Loop:
	for {
		v.Draw(ui)
		printFoot(t)
		ui.Flush()
		ev := event.PollEvent()
		switch ev := ev.(type) {
		case event.KeyPress:
			switch ev.Key {
			case 'j':
				v.MoveDown()
			case 'k':
				v.MoveUp()
			case 'h':
				v.MoveLeft()
			case 'l':
				v.MoveRight()
			case 'q':
				break Loop
			case 'u':
				t.Undo()
			case 'r':
				if ev.Ctrl {
					t.Redo()
				}

			case 'i':
				insertMode(v, t)
			}

		}
	}
}

func insertMode(v *view.View, t *text.Text) {
	for {
		_, h := ui.Size()
		printFoot(t)
		print(0, h-1, "-- INSERT --", console.AttrBold)
		ui.Flush()
		ev := event.PollEvent()
		switch ev := ev.(type) {
		case event.KeyPress:
			switch ev.Key {
			case event.Escape:
				t.CommitChanges()
				return
			case event.Backspace:
				v.MoveLeft()
				fallthrough
			case event.Delete:
				c := v.CurrentCell()
				length := utf8.RuneLen(c.R)
				t.Delete(c.Off, length)
			case event.Enter:
				t.Insert(v.CurrentCell().Off, []byte("\n"))
				v.ReadLines()
				v.MoveDown()
				v.ToTheStartColumn()
			default:
				buf := make([]byte, 4)
				n := utf8.EncodeRune(buf, rune(ev.Key))
				t.Insert(v.CurrentCell().Off, buf[:n])
				v.ReadLines()
				v.MoveRight()
			}
		}
		v.Draw(ui)
	}
}

func print(x, y int, s string, attrs uint8) {
	for _, r := range []rune(s) {
		ui.SetCell(x, y, r, attrs)
		x++
	}
}

func printFoot(t *text.Text) {
	w, h := ui.Size()
	for x := 0; x < w; x++ {
		ui.SetCell(x, h-2, ' ', console.AttrReverse|console.AttrBold)
	}
	filename := filename
	if t.Modified() {
		filename += " [+]"
	}
	print(0, h-2, filename, console.AttrReverse|console.AttrBold)
}
