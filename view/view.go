package view

import (
	"io"
	"unicode/utf8"

	"github.com/mibk/syd/core"
	"github.com/mibk/syd/ui"
)

const EOF = utf8.MaxRune + 1

type View struct {
	name string

	win ui.Window
	buf *core.Buffer

	origin int64
	q0, q1 int64
}

func New(win ui.Window, buf *core.Buffer) *View {
	return &View{win: win, buf: buf}
}

func (v *View) SetName(name string) { v.name = name }

// Size returns the size of v.
func (v *View) Size() (w, h int) { return v.win.Size() }

func (v *View) Position() (x, y int) { return v.win.Position() }

func (v *View) Frame() ui.Frame { return v.win.Body().Frame() }

func (v *View) Render() {
	v.LoadText()
	for _, r := range []rune(v.name) {
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
