package view

import (
	"io"

	"github.com/mibk/syd/core"
	"github.com/mibk/syd/ui/term"
)

const tabStop = 8

type View struct {
	width  int
	height int
	buf    *core.Buffer

	origin int64
	q0, q1 int64

	Frame Frame
}

func New(buf *core.Buffer) *View {
	return &View{buf: buf}
}

// Size returns the size of v.
func (v *View) Size() (w, h int) {
	return v.width, v.height
}

// SetSize sets the size of v.
func (v *View) SetSize(w, h int) {
	v.width, v.height = w, h
}

func (v *View) Render(ui term.UI) {
	attr := uint8(term.AttrDefault)
	selText := func(p int64, x, y int) {
		if p == v.q0 {
			if v.q0 == v.q1 {
				ui.SetCursor(x, y)
			} else {
				attr = term.AttrReverse
			}
		} else if p == v.q1 {
			attr = term.AttrDefault
		}
	}
	v.LoadText()
	ui.SetCursor(-1, -1)
	ui.Clear()
	p := v.origin
	for y, l := range v.Frame.Lines {
		x := 0
		for _, r := range l {
			selText(p, x, y)
			ui.SetCell(x, y, r, attr)
			p++
			if r == '\t' {
				x += TabWidthForCol(x)
			} else {
				x++
			}
		}
		selText(p, x, y)
		p++
	}
}

func (v *View) LoadText() {
	v.Frame.Lines = nil
	x, y := 0, 0
	p := v.origin
	for ; ; p++ {
		if len(v.Frame.Lines) <= y {
			v.Frame.Lines = append(v.Frame.Lines, nil)
		}
		if p == v.q0 {
			v.Frame.Line0, v.Frame.Col0 = y, x
			if v.Frame.WantCol == ColQ0 {
				v.Frame.WantCol = x
			}
		}
		if p == v.q1 {
			v.Frame.Line1, v.Frame.Col1 = y, x
			if v.Frame.WantCol == ColQ1 {
				v.Frame.WantCol = x
			}
		}
		r, _, err := v.buf.ReadRuneAt(p)
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
		if r != '\n' {
			v.Frame.Lines[y] = append(v.Frame.Lines[y], r)
		}

		if x >= v.width || r == '\n' {
			y++
			x = 0
			if y == v.height {
				break
			}
		} else if r == '\t' {
			x += TabWidthForCol(x)
		} else {
			x++
		}
	}
	v.Frame.Nchars = int(p - v.origin)
}

func (v *View) Origin() int64 { return v.origin }

func (v *View) SetOrigin(org int64) { v.origin = org }

func (v *View) Selected() (q0, q1 int64) { return v.q0, v.q1 }

func (v *View) Select(q0, q1 int64) {
	if q0 < 0 || q1 < q0 {
		return
	}
	v.q0, v.q1 = q0, q1
	if v.q1 > v.origin+int64(v.Frame.Nchars) {
		oldOrg := v.origin
		v.origin += int64(v.Frame.NextNewLine(3))
		v.LoadText()
		if v.q1 > v.origin+int64(v.Frame.Nchars) {
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
	v.q0, v.q1 = v.q0+1, v.q0+1
	v.Frame.WantCol = ColQ1
	v.checkVisibility()
}

func (v *View) DelSelected() {
	v.buf.Delete(v.q0, v.q1)
	v.q1 = v.q0
	v.checkVisibility()
}

func (v *View) checkVisibility() {
	if v.q0 < v.origin || v.q0 > v.origin+int64(v.Frame.Nchars)+1 {
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

func TabWidthForCol(col int) int {
	w := tabStop - col%tabStop
	if w == 0 {
		return tabStop
	}
	return w
}
