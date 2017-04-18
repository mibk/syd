package core

import (
	"io"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/mouse"

	"github.com/atotto/clipboard"
	"github.com/mibk/syd/ui"
	"github.com/mibk/syd/ui/term"
)

type Text struct {
	ctx  cmdContext
	text *term.Text
	buf  Buffer

	origin    int64
	q0, q1    int64
	selEnd    *int64
	timestamp time.Time
}

func newText(ctx cmdContext, buf Buffer, tt *term.Text) *Text {
	t := &Text{
		ctx:  ctx,
		buf:  buf,
		text: tt,
	}
	tt.OnMouseEvent(t.handleMouse)
	tt.OnKeyEvent(t.handleKeyEvent)
	return t
}

func (t *Text) loadText() {
	t.text.Select(int(t.q0-t.origin), int(t.q1-t.origin))

	for p := t.origin; ; p++ {
		r, _, err := t.buf.ReadRuneAt(p)
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
		if err := t.text.WriteRune(r); err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}

	}
}

func (t *Text) Origin() int64 { return t.origin }

func (t *Text) SetOrigin(org int64) { t.origin = org }

func (t *Text) Selected() (q0, q1 int64) { return t.q0, t.q1 }

func (t *Text) SelectionToString(q0, q1 int64) string {
	s := make([]rune, 0, q1-q0)
	for p := q0; p < q1; p++ {
		s = append(s, t.ReadRuneAt(p))
	}
	return string(s)
}

func (t *Text) Select(q0, q1 int64) {
	if q0 < 0 || q1 < q0 {
		return
	}
	t.q0, t.q1 = q0, q1
	if t.q1 > t.origin+int64(t.text.Frame().Nchars()) {
		oldOrg := t.origin
		t.origin += int64(t.text.Frame().CharsUntilXY(0, 3))
		t.loadText()
		if t.q1 > t.origin+int64(t.text.Frame().Nchars()) {
			// There's no more content, get back.
			t.origin = oldOrg
			t.q1--
			if t.q0 > t.q1 {
				t.q0 = t.q1
			}
			t.loadText()
		}
	}
	t.checkVisibility()
}

func (t *Text) Insert(s string) {
	if t.q0 != t.q1 {
		t.buf.Delete(t.q0, t.q1)
	}
	t.buf.Insert(t.q0, s)
	q := t.q0 + int64(utf8.RuneCountInString(s))
	t.q0, t.q1 = q, q
	t.text.Frame().SetWantCol(ui.ColQ1)
	t.checkVisibility()
}

func (t *Text) DeleteSel() {
	t.buf.Delete(t.q0, t.q1)
	t.q1 = t.q0
	t.checkVisibility()
}

func (t *Text) checkVisibility() {
	if t.q0 < t.origin || t.q0 > t.origin+int64(t.text.Frame().Nchars())+1 {
		t.origin = t.PrevNewLine(t.q0, 3)
	}
}

func (t *Text) PrevNewLine(p int64, n int) int64 {
	for ; n > 0; n-- {
		// Shorten long lines. After 128 characters call it a line anyway.
		for i := 0; i < 128 && p > 0; i++ {
			p--
			if p == 0 {
				return 0
			}
			r, _, err := t.buf.ReadRuneAt(p - 1)
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

func (t *Text) ReadRuneAt(off int64) rune {
	r, _, err := t.buf.ReadRuneAt(off)
	if err == io.EOF {
		return EOF
	} else if err != nil {
		panic(err)
	}
	return r
}

func (t *Text) handleMouse(p int, ev mouse.Event) {
	q := t.origin + int64(p)
	switch ev.Direction {
	case mouse.DirPress:
		if ev.Button == mouse.ButtonMiddle {
			cmd := t.SelectionToString(t.dblclick(q))
			execute(t.ctx, cmd)
			return
		} else if ev.Button == mouse.ButtonRight {
			path := t.SelectionToString(t.selectPath(q))
			// TODO: Don't require being in a column context. Just
			// open the file in the most recent column (using similar
			// heuristic as in Acme).
			if col, ok := t.ctx.column(); ok {
				col.NewWindowFile(path)
			}
			return
		}

		if time.Since(t.timestamp) < 300*time.Millisecond {
			t.Select(t.dblclick(q))
			t.selEnd = nil
			return
		}
		t.q0, t.q1 = q, q
		t.selEnd = &t.q1
		t.timestamp = time.Now()
		// TODO: Get rid of SetWantCol.
		t.text.Frame().SetWantCol(ui.ColQ0)
	case mouse.DirRelease:
		t.selEnd = nil
	case mouse.DirNone:
		if t.selEnd == nil {
			return
		}
		*t.selEnd = q
		if t.q0 > t.q1 {
			t.q0, t.q1 = t.q1, t.q0
			if t.selEnd == &t.q0 {
				t.selEnd = &t.q1
			} else {
				t.selEnd = &t.q0
			}
		}
	case mouse.DirStep:
		switch ev.Button {
		case mouse.ButtonWheelUp:
			t.ScrollUp(3)
		case mouse.ButtonWheelDown:
			t.ScrollDown(3)
		}
	}
}

func (t *Text) dblclick(q int64) (q0, q1 int64) {
	return t.spread(q, isAlphaNumeric)
}

func (t *Text) selectPath(q int64) (q0, q1 int64) {
	return t.spread(q, isPath)
}

func (t *Text) spread(q int64, fn func(rune) bool) (q0, q1 int64) {
	q0, q1 = q, q
	for q0 > 0 {
		r := t.ReadRuneAt(q0 - 1)
		if !fn(r) {
			break
		}
		q0--
	}
	for {
		r := t.ReadRuneAt(q1)
		if !fn(r) {
			break
		}
		q1++
	}
	return
}

func isAlphaNumeric(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

func isPath(r rune) bool { return !unicode.IsSpace(r) }

func (t *Text) handleKeyEvent(ev key.Event) {
	switch {
	case ev.Rune == ui.KeyEnter:
		q0, _ := t.Selected()
		p := t.PrevNewLine(q0, 1)

		var indent []rune
		for ; ; p++ {
			r := t.ReadRuneAt(p)
			if r != ' ' && r != '\t' {
				break
			}
			indent = append(indent, r)
		}
		t.Insert("\n" + string(indent))
	case ev.Rune == ui.KeyBackspace:
		q0, q1 := t.Selected()
		if q0 == q1 {
			t.Select(q0-1, q1)
		}
		t.DeleteSel()
	case ev.Rune == ui.KeyDelete:
		q0, q1 := t.Selected()
		if q0 == q1 {
			t.Select(q0, q1+1)
		}
		t.DeleteSel()
	case ev.Rune == ui.KeyEscape:
		t.DeleteSel()
	case ev.Rune == ui.KeyLeft:
		left(t)
	case ev.Rune == ui.KeyRight:
		right(t)

	case ev.Rune == ui.KeyUp:
		up(t)
	case ev.Rune == ui.KeyDown:
		down(t)

	case (ev.Rune == 'c' || ev.Rune == 'x') && ev.Modifiers&key.ModControl != 0:
		err := clipboard.WriteAll(t.SelectionToString(t.Selected()))
		if err != nil {
			panic(err)
		}
		if ev.Rune == 'x' {
			t.DeleteSel()
		}
	case ev.Rune == 'v' && ev.Modifiers&key.ModControl != 0:
		s, err := clipboard.ReadAll()
		if err != nil {
			panic(err)
		}
		t.Insert(s)
	default:
		t.Insert(string(ev.Rune))
	}
}

// TODO: Remove these.

func (t *Text) ScrollUp(nlines int) {
	t.SetOrigin(t.PrevNewLine(t.Origin(), nlines))
}

func (t *Text) ScrollDown(nlines int) {
	t.SetOrigin(t.Origin() + int64(t.text.Frame().CharsUntilXY(0, nlines)))
}

// TODO: Is this the right place for these?

func left(t *Text) {
	q0, _ := t.Selected()
	t.Select(q0-1, q0-1)
	t.text.Frame().SetWantCol(ui.ColQ0)
}

func right(t *Text) {
	_, q1 := t.Selected()
	t.Select(q1+1, q1+1)
	t.text.Frame().SetWantCol(ui.ColQ1)
}

func up(t *Text) {
	_, line1 := t.text.Frame().SelectionLines()
	q := findQ(t, line1-1)
	t.Select(q, q)
}

func down(t *Text) {
	_, line1 := t.text.Frame().SelectionLines()
	q := findQ(t, line1+1)
	t.Select(q, q)
}

func findQ(t *Text, line int) int64 {
	if line < 0 {
		t.SetOrigin(t.PrevNewLine(t.Origin(), -line))
		t.loadText()
		line = 0
	} else if line > t.text.Frame().Lines()-1 {
		_, h := t.text.Size()
		if t.text.Frame().Lines() == h {
			i := line - t.text.Frame().Lines() + 1
			oldOrg := t.Origin()
			l := t.text.Frame().Lines()
			t.SetOrigin(oldOrg + int64(t.text.Frame().CharsUntilXY(0, i)))
			t.loadText()
			if t.text.Frame().Lines() < l {
				t.SetOrigin(oldOrg)
				t.loadText()
			}
		}
		line = t.text.Frame().Lines() - 1
	}
	q := t.Origin()
	return q + int64(t.text.Frame().CharsUntilXY(t.text.Frame().WantCol(), line))
}
