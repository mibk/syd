package main

import (
	"unicode"

	"github.com/mibk/syd/ui"
	"github.com/mibk/syd/view"
)

var (
	qvis    = -1
	linevis = -1
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
		qvis = -1
	case ev.Key == ui.KeyDelete:
		q0, q1 := v.Selected()
		if q0 == q1 {
			v.Select(q0, q1+1)
		}
		v.DelSelected()
		qvis = -1
	case ev.Key == ui.KeyLeft:
		left(v)
	case ev.Key == ui.KeyRight:
		right(v)

	case ev.Key == ui.KeyUp:
		up(v)
	case ev.Key == ui.KeyDown:
		down(v)

	case ev.Key == 'v' && ev.Ctrl:
		visualMode(v)

	default:
		v.Insert(string(ev.Key))
		qvis = -1
	}
}

func left(v *view.View) {
	if qvis == -1 {
		q0, _ := v.Selected()
		v.Select(q0-1, q0-1)
	} else {
		vismove(v, -1)
	}
	v.Frame.WantCol = view.ColQ0
}

func right(v *view.View) {
	if qvis == -1 {
		_, q1 := v.Selected()
		v.Select(q1+1, q1+1)
	} else {
		vismove(v, +1)
	}
	v.Frame.WantCol = view.ColQ1
}

func up(v *view.View) {
	if qvis == -1 {
		q := findQ(v, v.Frame.Line1-1)
		v.Select(q, q)
	} else {
		qv, line := visQAndLine(v)
		q := findQ(v, line-1)
		vismove(v, q-qv)
	}
}

func down(v *view.View) {
	if qvis == -1 {
		q := findQ(v, v.Frame.Line1+1)
		v.Select(q, q)
	} else {
		qv, line := visQAndLine(v)
		q := findQ(v, line+1)
		vismove(v, q-qv)
	}
}

func findQ(v *view.View, line int) int64 {
	if line < 0 {
		v.SetOrigin(v.PrevNewLine(v.Origin(), -line))
		v.LoadText()
		line = 0
	} else if line > len(v.Frame.Lines)-1 {
		_, h := v.Size()
		if len(v.Frame.Lines) == h {
			i := line - len(v.Frame.Lines) + 1
			oldOrg := v.Origin()
			l := len(v.Frame.Lines)
			v.SetOrigin(oldOrg + int64(v.Frame.NextNewLine(i)))
			v.LoadText()
			if len(v.Frame.Lines) < l {
				v.SetOrigin(oldOrg)
				v.LoadText()
			}
		}
		line = len(v.Frame.Lines) - 1
	}
	q := v.Origin()
	for n, l := range v.Frame.Lines {
		if n < line {
			q += int64(len(l)) + 1 // + '\n'
			continue
		}
		return q + int64(view.CharsToX(v.Frame.Lines[n], v.Frame.WantCol))
	}
	panic("shouldn't happen")
}

func visQAndLine(v *view.View) (q int64, line int) {
	q0, q1 := v.Selected()
	if qvis == 0 {
		return q0, v.Frame.Line0
	}
	return q1, v.Frame.Line1
}

func vismove(v *view.View, d int64) {
	q0, q1 := v.Selected()
	var q *int64
	if qvis == 0 {
		q = &q0
	} else {
		q = &q1
	}
	*q += d
	if q1 < q0 {
		q0, q1 = q1, q0
		if qvis == 0 {
			qvis = 1
			linevis = 1
		} else {
			qvis = 0
			linevis = 0
		}
	}
	v.Select(q0, q1)
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
	v.SetOrigin(v.Origin() + int64(v.Frame.CharsToXY(0, nlines)))
}

func visualMode(v *view.View) {
	if qvis == -1 {
		qvis = 0
		linevis = 0
	} else {
		qvis = -1
	}
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
