package main

import (
	"unicode"

	"github.com/mibk/syd/ui"
	"github.com/mibk/syd/view"
)

func handleKeyPress(v *view.View, ev ui.KeyPress) {
	switch {
	case ev.Key == ui.KeyEscape:
	case ev.Key == ui.KeyBackspace:
		q0, q1 := v.Selected()
		if q0 == q1 {
			v.Select(q0-1, q1)
		}
		v.DelSelected()
	case ev.Key == ui.KeyDelete:
		q0, q1 := v.Selected()
		if q0 == q1 {
			v.Select(q0, q1+1)
		}
		v.DelSelected()
	case ev.Key == ui.KeyLeft:
		left(v)
	case ev.Key == ui.KeyRight:
		right(v)

	case ev.Key == ui.KeyUp:
		up(v)
	case ev.Key == ui.KeyDown:
		down(v)

	default:
		v.Insert(string(ev.Key))
	}
}

func left(v *view.View) {
	q0, _ := v.Selected()
	v.Select(q0-1, q0-1)
	v.Frame().SetWantCol(ui.ColQ0)
}

func right(v *view.View) {
	_, q1 := v.Selected()
	v.Select(q1+1, q1+1)
	v.Frame().SetWantCol(ui.ColQ1)
}

func up(v *view.View) {
	_, line1 := v.Frame().SelectionLines()
	q := findQ(v, line1-1)
	v.Select(q, q)
}

func down(v *view.View) {
	_, line1 := v.Frame().SelectionLines()
	q := findQ(v, line1+1)
	v.Select(q, q)
}

func findQ(v *view.View, line int) int64 {
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

func pageUp(v *view.View) {
	_, h := v.Size()
	scrollUp(v, h)
}

func scrollUp(v *view.View, nlines int) {
	v.SetOrigin(v.PrevNewLine(v.Origin(), nlines))
}

func pageDown(v *view.View) {
	_, h := v.Size()
	scrollDown(v, h)
}

func scrollDown(v *view.View, nlines int) {
	v.SetOrigin(v.Origin() + int64(v.Frame().CharsUntilXY(0, nlines)))
}

func dblclick(v *view.View, q int64) (q0, q1 int64) {
	q0, q1 = q, q
	for q0 > 0 {
		r, err := v.ReadRuneAt(q0 - 1)
		if err != nil || !isAlphaNumeric(r) {
			break
		}
		q0--
	}
	for {
		r, err := v.ReadRuneAt(q1)
		if err != nil || !isAlphaNumeric(r) {
			break
		}
		q1++
	}
	return
}

func isAlphaNumeric(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }
