package core

import (
	"io"
	"os"
	"unicode"
	"unicode/utf8"

	"github.com/mibk/syd/ui"
)

type Text struct {
	ctx  cmdContext
	text ui.Text
	buf  Buffer

	origin int64
	q0, q1 int64
	selEnd *int64

	// position for ReadRune
	pp int64
}

func newText(ctx cmdContext, buf Buffer, tt ui.Text) *Text {
	t := &Text{
		ctx:  ctx,
		buf:  buf,
		text: tt,
	}
	tt.Init(t)
	return t
}

func (t *Text) Reset() {
	t.pp = t.origin
}

func (t *Text) ReadRune() (r rune, size int, err error) {
	t.pp++
	return t.buf.ReadRuneAt(t.pp - 1)
}

func (t *Text) redraw() {
	if err := t.text.Reload(); err != nil {
		panic(err)
	}
}

func (t *Text) Origin() int64 { return t.origin }

func (t *Text) SetOrigin(org int64) { t.origin = org }

func (t *Text) Selected() (q0, q1 int64) { return t.q0, t.q1 }

func (t *Text) SelectionToString(q0, q1 int64) string {
	s := make([]rune, 0, q1-q0)
	for p := q0; p < q1; p++ {
		s = append(s, t.readRuneAt(p))
	}
	return string(s)
}

func (t *Text) Select(q0, q1 int64) {
	if q0 < 0 || q1 < q0 {
		return
	}
	t.q0, t.q1 = q0, q1
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
	t.redraw()
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

func (t *Text) readRuneAt(off int64) rune {
	r, _, err := t.buf.ReadRuneAt(off)
	if err == io.EOF {
		return EOF
	} else if err != nil {
		panic(err)
	}
	return r
}

func (t *Text) StartSel(q int64) {
	t.q0, t.q1 = q, q
	t.selEnd = &t.q1
	// TODO: Get rid of SetWantCol.
	t.text.Frame().SetWantCol(ui.ColQ0)
}

func (t *Text) MoveSel(q int64) {
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
}

func (t *Text) StopSel() { t.selEnd = nil }

func (t *Text) SelectUnderCursor(q int64) {
	t.Select(t.dblclick(q))
	t.selEnd = nil
}

func (t *Text) ExecuteUnderCursor(q int64) {
	cmd := t.selected(q)
	if cmd == "" {
		cmd = t.selectPath(q)
	}
	execute(t.ctx, cmd)
}

func (t *Text) Plumb(q int64) {
	query := t.selected(q)

	// TODO: Don't require being in the column context. Just
	// open the file in the most recent column (using similar
	// heuristic as in Acme).
	if col, ok := t.ctx.column(); ok {
		path := query
		if path == "" {
			path = t.selectPath(q)
		}
		if _, ok := col.ed.wins[path]; ok {
			return
		}
		if _, err := os.Stat(path); err == nil {
			col.NewWindowFile(path)
			return
		}
	}

	if win, ok := t.ctx.window(); ok {
		if query == "" {
			q0, q1 := t.dblclick(q)
			t.Select(q0, q1)
			query = t.SelectionToString(q0, q1)
		}
		win.findNextExactMatch(query)
	}
}

// selected returns the selection if q is between
// t.q0 and t.q1, otherwise it returns an empty
// string.
func (t *Text) selected(q int64) string {
	if q >= t.q0 && q < t.q1 {
		return t.SelectionToString(t.q0, t.q1)
	}
	return ""
}

func (t *Text) dblclick(q int64) (q0, q1 int64) {
	return t.spread(q, isAlphaNumeric)
}

func (t *Text) selectPath(q int64) string {
	return t.SelectionToString(t.spread(q, isPath))
}

func (t *Text) spread(q int64, fn func(rune) bool) (q0, q1 int64) {
	q0, q1 = q, q
	for q0 > 0 {
		r := t.readRuneAt(q0 - 1)
		if !fn(r) {
			break
		}
		q0--
	}
	for {
		r := t.readRuneAt(q1)
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

func isPath(r rune) bool { return !unicode.IsSpace(r) && r != EOF && r != 0 }

func (t *Text) InsertNewLine() {
	q0, _ := t.Selected()
	p := t.PrevNewLine(q0, 1)

	var indent []rune
	for ; ; p++ {
		r := t.readRuneAt(p)
		if r != ' ' && r != '\t' {
			break
		}
		indent = append(indent, r)
	}
	t.Insert("\n" + string(indent))
}

func (t *Text) ScrollUp(nlines int) {
	t.SetOrigin(t.PrevNewLine(t.Origin(), nlines))
}

func (t *Text) ScrollDown(nlines int) {
	t.SetOrigin(t.Origin() + int64(t.text.Frame().CharsUntilXY(0, nlines)))
}

// TODO: Remove these.

func (t *Text) Up() {
	_, line1 := t.text.Frame().SelectionLines()
	q := findQ(t, line1-1)
	t.Select(q, q)
}

func (t *Text) Down() {
	_, line1 := t.text.Frame().SelectionLines()
	q := findQ(t, line1+1)
	t.Select(q, q)
}

func findQ(t *Text, line int) int64 {
	if line < 0 {
		t.SetOrigin(t.PrevNewLine(t.Origin(), -line))
		t.redraw()
		line = 0
	} else if line > t.text.Frame().Lines()-1 {
		_, h := t.text.Size()
		if t.text.Frame().Lines() == h {
			i := line - t.text.Frame().Lines() + 1
			oldOrg := t.Origin()
			l := t.text.Frame().Lines()
			t.SetOrigin(oldOrg + int64(t.text.Frame().CharsUntilXY(0, i)))
			t.redraw()
			if t.text.Frame().Lines() < l {
				t.SetOrigin(oldOrg)
				t.redraw()
			}
		}
		line = t.text.Frame().Lines() - 1
	}
	q := t.Origin()
	return q + int64(t.text.Frame().CharsUntilXY(t.text.Frame().WantCol(), line))
}
