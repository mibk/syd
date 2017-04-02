package view

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

	"github.com/mibk/syd/core"
	"github.com/mibk/syd/ui"
	"github.com/mibk/syd/ui/term"
)

const EOF = utf8.MaxRune + 1

type View struct {
	filename string

	win *term.Window
	buf *core.Buffer

	origin    int64
	q0, q1    int64
	pressed   bool
	timestamp time.Time
}

func New(win *term.Window, buf *core.Buffer) *View {
	v := &View{win: win, buf: buf}
	v.win.Body().OnMouseEvent(v.handleMouse)
	v.win.Body().OnKeyEvent(v.handleKeyEvent)
	return v
}

func (v *View) SetFilename(filename string) { v.filename = filename }

// Size returns the size of v.
func (v *View) Size() (w, h int) { return v.win.Size() }

func (v *View) Frame() *term.Frame { return v.win.Body().Frame() }

func (v *View) Render() {
	v.LoadText()
	for _, r := range []rune(v.filename) {
		v.win.Head().WriteRune(r)
	}
	v.win.Flush()
}

func (v *View) LoadText() {
	v.win.Clear()
	v.win.Body().Select(int(v.q0-v.origin), int(v.q1-v.origin))

	for p := v.origin; ; p++ {
		r, _, err := v.buf.ReadRuneAt(p)
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
		if err := v.win.Body().WriteRune(r); err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}

	}
}

func (v *View) Origin() int64 { return v.origin }

func (v *View) SetOrigin(org int64) { v.origin = org }

func (v *View) Selected() (q0, q1 int64) { return v.q0, v.q1 }

func (v *View) Select(q0, q1 int64) {
	if q0 < 0 || q1 < q0 {
		return
	}
	v.q0, v.q1 = q0, q1
	if v.q1 > v.origin+int64(v.Frame().Nchars()) {
		oldOrg := v.origin
		v.origin += int64(v.Frame().CharsUntilXY(0, 3))
		v.LoadText()
		if v.q1 > v.origin+int64(v.Frame().Nchars()) {
			// There's no more content, get back.
			v.origin = oldOrg
			v.q1--
			if v.q0 > v.q1 {
				v.q0 = v.q1
			}
			v.LoadText()
		}
	}
	v.checkVisibility()
}

func (v *View) Insert(s string) {
	if v.q0 != v.q1 {
		v.buf.Delete(v.q0, v.q1)
	}
	v.buf.Insert(v.q0, s)
	q := v.q0 + int64(utf8.RuneCountInString(s))
	v.q0, v.q1 = q, q
	v.Frame().SetWantCol(ui.ColQ1)
	v.checkVisibility()
}

func (v *View) DeleteSel() {
	v.buf.Delete(v.q0, v.q1)
	v.q1 = v.q0
	v.checkVisibility()
}

func (v *View) checkVisibility() {
	if v.q0 < v.origin || v.q0 > v.origin+int64(v.Frame().Nchars())+1 {
		v.origin = v.PrevNewLine(v.q0, 3)
	}
}

func (v *View) Undo() { v.buf.Undo() }
func (v *View) Redo() { v.buf.Redo() }

func (v *View) PrevNewLine(p int64, n int) int64 {
	for ; n > 0; n-- {
		// Shorten long lines. After 128 characters call it a line anyway.
		for i := 0; i < 128 && p > 0; i++ {
			p--
			if p == 0 {
				return 0
			}
			r, _, err := v.buf.ReadRuneAt(p - 1)
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

func (v *View) ReadRuneAt(off int64) rune {
	r, _, err := v.buf.ReadRuneAt(off)
	if err == io.EOF {
		return EOF
	} else if err != nil {
		panic(err)
	}
	return r
}

////////////////

func (v *View) handleMouse(p int, ev mouse.Event) {
	q := v.origin + int64(p)
	switch ev.Direction {
	case mouse.DirPress:
		if ev.Button == mouse.ButtonMiddle {
			q0, q1 := v.dblclick(q)
			var cmd []rune
			for i := q0; i < q1; i++ {
				cmd = append(cmd, v.ReadRuneAt(i))
			}
			v.execute(string(cmd))
			return
		} else if ev.Button == mouse.ButtonRight {
			return
		}

		if time.Since(v.timestamp) < 300*time.Millisecond {
			v.Select(v.dblclick(q))
			v.pressed = false
			return
		}
		v.q0, v.q1 = q, q
		v.pressed = true
		v.timestamp = time.Now()
		// TODO: Get rid of SetWantCol.
		v.Frame().SetWantCol(ui.ColQ0)
	case mouse.DirRelease:
		v.pressed = false
	case mouse.DirNone:
		if !v.pressed {
			return
		}
		v.q1 = q
		if v.q0 > v.q1 {
			v.q0, v.q1 = v.q1, v.q0
		}
	case mouse.DirStep:
		switch ev.Button {
		case mouse.ButtonWheelUp:
			v.ScrollUp(3)
		case mouse.ButtonWheelDown:
			v.ScrollDown(3)
		}
	}
}

func (v *View) dblclick(q int64) (q0, q1 int64) {
	q0, q1 = q, q
	for q0 > 0 {
		r := v.ReadRuneAt(q0 - 1)
		if !isAlphaNumeric(r) {
			break
		}
		q0--
	}
	for {
		r := v.ReadRuneAt(q1)
		if !isAlphaNumeric(r) {
			break
		}
		q1++
	}
	return
}

func (v *View) execute(command string) {
	switch command {
	case "Exit":
		// TODO: This is just a temporary solution
		// until a proper solution is found.
		go func() {
			ui.Events <- ui.Quit
		}()
	case "Put":
		if v.filename != "" {
			if err := v.saveFile(); err != nil {
				panic(err)
			}
		}
	case "Undo":
		v.Undo()
	case "Redo":
		v.Redo()
	default:
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

func (v *View) handleKeyEvent(ev key.Event) {
	switch {
	case ev.Rune == ui.KeyEnter:
		q0, _ := v.Selected()
		p := v.PrevNewLine(q0, 1)

		var indent []rune
		for ; ; p++ {
			r := v.ReadRuneAt(p)
			if r != ' ' && r != '\t' {
				break
			}
			indent = append(indent, r)
		}
		v.Insert("\n" + string(indent))
	case ev.Rune == ui.KeyBackspace:
		q0, q1 := v.Selected()
		if q0 == q1 {
			v.Select(q0-1, q1)
		}
		v.DeleteSel()
	case ev.Rune == ui.KeyDelete:
		q0, q1 := v.Selected()
		if q0 == q1 {
			v.Select(q0, q1+1)
		}
		v.DeleteSel()
	case ev.Rune == ui.KeyLeft:
		left(v)
	case ev.Rune == ui.KeyRight:
		right(v)

	case ev.Rune == ui.KeyUp:
		up(v)
	case ev.Rune == ui.KeyDown:
		down(v)

	default:
		v.Insert(string(ev.Rune))
	}
}

func (v *View) saveFile() error {
	// TODO: Read bytes directly from the undo.Buffer.
	// TODO: Don't use '~' suffix, make saving safer.
	f, err := os.Create(v.filename + "~")
	if err != nil {
		return err
	}

	var buf [64]byte
	var i int

	for p := int64(0); ; p++ {
		r := v.ReadRuneAt(p)
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

	return os.Rename(v.filename+"~", v.filename)
}

func isAlphaNumeric(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

// TODO: Remove these.

func (v *View) ScrollUp(nlines int) {
	v.SetOrigin(v.PrevNewLine(v.Origin(), nlines))
}

func (v *View) ScrollDown(nlines int) {
	v.SetOrigin(v.Origin() + int64(v.Frame().CharsUntilXY(0, nlines)))
}

////////////////

// TODO: Is this the right place for these?

func left(v *View) {
	q0, _ := v.Selected()
	v.Select(q0-1, q0-1)
	v.Frame().SetWantCol(ui.ColQ0)
}

func right(v *View) {
	_, q1 := v.Selected()
	v.Select(q1+1, q1+1)
	v.Frame().SetWantCol(ui.ColQ1)
}

func up(v *View) {
	_, line1 := v.Frame().SelectionLines()
	q := findQ(v, line1-1)
	v.Select(q, q)
}

func down(v *View) {
	_, line1 := v.Frame().SelectionLines()
	q := findQ(v, line1+1)
	v.Select(q, q)
}

func findQ(v *View, line int) int64 {
	if line < 0 {
		v.SetOrigin(v.PrevNewLine(v.Origin(), -line))
		v.LoadText()
		line = 0
	} else if line > v.Frame().Lines()-1 {
		_, h := v.Size()
		if v.Frame().Lines() == h {
			i := line - v.Frame().Lines() + 1
			oldOrg := v.Origin()
			l := v.Frame().Lines()
			v.SetOrigin(oldOrg + int64(v.Frame().CharsUntilXY(0, i)))
			v.LoadText()
			if v.Frame().Lines() < l {
				v.SetOrigin(oldOrg)
				v.LoadText()
			}
		}
		line = v.Frame().Lines() - 1
	}
	q := v.Origin()
	return q + int64(v.Frame().CharsUntilXY(v.Frame().WantCol(), line))
}
