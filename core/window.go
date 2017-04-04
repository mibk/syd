package core

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/mouse"

	"github.com/mibk/syd/ui"
	"github.com/mibk/syd/ui/term"
)

const EOF = utf8.MaxRune + 1

type Window struct {
	filename string

	win *term.Window
	buf *Buffer

	origin    int64
	q0, q1    int64
	pressed   bool
	timestamp time.Time
}

func NewWindow(window *term.Window, buf *Buffer) *Window {
	win := &Window{win: window, buf: buf}
	win.win.Body().OnMouseEvent(win.handleMouse)
	win.win.Body().OnKeyEvent(win.handleKeyEvent)
	return win
}

func (win *Window) SetFilename(filename string) { win.filename = filename }

// Size returns the size of win.
func (win *Window) Size() (w, h int) { return win.win.Size() }

func (win *Window) Frame() *term.Frame { return win.win.Body().Frame() }

func (win *Window) Render() {
	win.LoadText()
	for _, r := range []rune(win.filename) {
		win.win.Head().WriteRune(r)
	}
	win.win.Flush()
}

func (win *Window) LoadText() {
	win.win.Clear()
	win.win.Body().Select(int(win.q0-win.origin), int(win.q1-win.origin))

	for p := win.origin; ; p++ {
		r, _, err := win.buf.ReadRuneAt(p)
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
		if err := win.win.Body().WriteRune(r); err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}

	}
}

func (win *Window) Origin() int64 { return win.origin }

func (win *Window) SetOrigin(org int64) { win.origin = org }

func (win *Window) Selected() (q0, q1 int64) { return win.q0, win.q1 }

func (win *Window) Select(q0, q1 int64) {
	if q0 < 0 || q1 < q0 {
		return
	}
	win.q0, win.q1 = q0, q1
	if win.q1 > win.origin+int64(win.Frame().Nchars()) {
		oldOrg := win.origin
		win.origin += int64(win.Frame().CharsUntilXY(0, 3))
		win.LoadText()
		if win.q1 > win.origin+int64(win.Frame().Nchars()) {
			// There's no more content, get back.
			win.origin = oldOrg
			win.q1--
			if win.q0 > win.q1 {
				win.q0 = win.q1
			}
			win.LoadText()
		}
	}
	win.checkVisibility()
}

func (win *Window) Insert(s string) {
	if win.q0 != win.q1 {
		win.buf.Delete(win.q0, win.q1)
	}
	win.buf.Insert(win.q0, s)
	q := win.q0 + int64(utf8.RuneCountInString(s))
	win.q0, win.q1 = q, q
	win.Frame().SetWantCol(ui.ColQ1)
	win.checkVisibility()
}

func (win *Window) DeleteSel() {
	win.buf.Delete(win.q0, win.q1)
	win.q1 = win.q0
	win.checkVisibility()
}

func (win *Window) checkVisibility() {
	if win.q0 < win.origin || win.q0 > win.origin+int64(win.Frame().Nchars())+1 {
		win.origin = win.PrevNewLine(win.q0, 3)
	}
}

func (win *Window) Undo() { win.buf.Undo() }
func (win *Window) Redo() { win.buf.Redo() }

func (win *Window) PrevNewLine(p int64, n int) int64 {
	for ; n > 0; n-- {
		// Shorten long lines. After 128 characters call it a line anyway.
		for i := 0; i < 128 && p > 0; i++ {
			p--
			if p == 0 {
				return 0
			}
			r, _, err := win.buf.ReadRuneAt(p - 1)
			if err != nil {
				panic(err)
			}
			if r == '\n' {
				break
			}
		}
	}
	return p
}

func (win *Window) ReadRuneAt(off int64) rune {
	r, _, err := win.buf.ReadRuneAt(off)
	if err == io.EOF {
		return EOF
	} else if err != nil {
		panic(err)
	}
	return r
}

////////////////

func (win *Window) handleMouse(p int, ev mouse.Event) {
	q := win.origin + int64(p)
	switch ev.Direction {
	case mouse.DirPress:
		if ev.Button == mouse.ButtonMiddle {
			q0, q1 := win.dblclick(q)
			var cmd []rune
			for i := q0; i < q1; i++ {
				cmd = append(cmd, win.ReadRuneAt(i))
			}
			win.execute(string(cmd))
			return
		} else if ev.Button == mouse.ButtonRight {
			return
		}

		if time.Since(win.timestamp) < 300*time.Millisecond {
			win.Select(win.dblclick(q))
			win.pressed = false
			return
		}
		win.q0, win.q1 = q, q
		win.pressed = true
		win.timestamp = time.Now()
		// TODO: Get rid of SetWantCol.
		win.Frame().SetWantCol(ui.ColQ0)
	case mouse.DirRelease:
		win.pressed = false
	case mouse.DirNone:
		if !win.pressed {
			return
		}
		win.q1 = q
		if win.q0 > win.q1 {
			win.q0, win.q1 = win.q1, win.q0
		}
	case mouse.DirStep:
		switch ev.Button {
		case mouse.ButtonWheelUp:
			win.ScrollUp(3)
		case mouse.ButtonWheelDown:
			win.ScrollDown(3)
		}
	}
}

func (win *Window) dblclick(q int64) (q0, q1 int64) {
	q0, q1 = q, q
	for q0 > 0 {
		r := win.ReadRuneAt(q0 - 1)
		if !isAlphaNumeric(r) {
			break
		}
		q0--
	}
	for {
		r := win.ReadRuneAt(q1)
		if !isAlphaNumeric(r) {
			break
		}
		q1++
	}
	return
}

func (win *Window) execute(command string) {
	switch command {
	case "Exit":
		// TODO: This is just a temporary solution
		// until a proper solution is found.
		go func() {
			ui.Events <- ui.Quit
		}()
	case "Put":
		if win.filename != "" {
			if err := win.saveFile(); err != nil {
				panic(err)
			}
		}
	case "Undo":
		win.Undo()
	case "Redo":
		win.Redo()
	default:
		var selected []rune
		q0, q1 := win.Selected()
		for p := q0; p < q1; p++ {
			r := win.ReadRuneAt(p)
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
		win.Insert(s)
		win.Select(q0, q0+int64(utf8.RuneCountInString(s)))
	}
}

func (win *Window) handleKeyEvent(ev key.Event) {
	switch {
	case ev.Rune == ui.KeyEnter:
		q0, _ := win.Selected()
		p := win.PrevNewLine(q0, 1)

		var indent []rune
		for ; ; p++ {
			r := win.ReadRuneAt(p)
			if r != ' ' && r != '\t' {
				break
			}
			indent = append(indent, r)
		}
		win.Insert("\n" + string(indent))
	case ev.Rune == ui.KeyBackspace:
		q0, q1 := win.Selected()
		if q0 == q1 {
			win.Select(q0-1, q1)
		}
		win.DeleteSel()
	case ev.Rune == ui.KeyDelete:
		q0, q1 := win.Selected()
		if q0 == q1 {
			win.Select(q0, q1+1)
		}
		win.DeleteSel()
	case ev.Rune == ui.KeyLeft:
		left(win)
	case ev.Rune == ui.KeyRight:
		right(win)

	case ev.Rune == ui.KeyUp:
		up(win)
	case ev.Rune == ui.KeyDown:
		down(win)

	default:
		win.Insert(string(ev.Rune))
	}
}

func (win *Window) saveFile() error {
	// TODO: Read bytes directly from the undo.Buffer.
	// TODO: Don't use '~' suffix, make saving safer.
	f, err := os.Create(win.filename + "~")
	if err != nil {
		return err
	}

	var buf [64]byte
	var i int

	for p := int64(0); ; p++ {
		r := win.ReadRuneAt(p)
		if r == EOF || len(buf[i:]) < utf8.UTFMax {
			if _, err := f.Write(buf[:i]); err != nil {
				return err
			}
			i = 0
		}
		if r == EOF {
			break
		}
		i += utf8.EncodeRune(buf[i:], r)
	}
	f.Close()

	return os.Rename(win.filename+"~", win.filename)
}

func isAlphaNumeric(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

// TODO: Remove these.

func (win *Window) ScrollUp(nlines int) {
	win.SetOrigin(win.PrevNewLine(win.Origin(), nlines))
}

func (win *Window) ScrollDown(nlines int) {
	win.SetOrigin(win.Origin() + int64(win.Frame().CharsUntilXY(0, nlines)))
}

////////////////

// TODO: Is this the right place for these?

func left(win *Window) {
	q0, _ := win.Selected()
	win.Select(q0-1, q0-1)
	win.Frame().SetWantCol(ui.ColQ0)
}

func right(win *Window) {
	_, q1 := win.Selected()
	win.Select(q1+1, q1+1)
	win.Frame().SetWantCol(ui.ColQ1)
}

func up(win *Window) {
	_, line1 := win.Frame().SelectionLines()
	q := findQ(win, line1-1)
	win.Select(q, q)
}

func down(win *Window) {
	_, line1 := win.Frame().SelectionLines()
	q := findQ(win, line1+1)
	win.Select(q, q)
}

func findQ(win *Window, line int) int64 {
	if line < 0 {
		win.SetOrigin(win.PrevNewLine(win.Origin(), -line))
		win.LoadText()
		line = 0
	} else if line > win.Frame().Lines()-1 {
		_, h := win.Size()
		if win.Frame().Lines() == h {
			i := line - win.Frame().Lines() + 1
			oldOrg := win.Origin()
			l := win.Frame().Lines()
			win.SetOrigin(oldOrg + int64(win.Frame().CharsUntilXY(0, i)))
			win.LoadText()
			if win.Frame().Lines() < l {
				win.SetOrigin(oldOrg)
				win.LoadText()
			}
		}
		line = win.Frame().Lines() - 1
	}
	q := win.Origin()
	return q + int64(win.Frame().CharsUntilXY(win.Frame().WantCol(), line))
}
