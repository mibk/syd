package main

import (
	"github.com/mibk/syd/ui"
	"github.com/mibk/syd/view"
)

func handleKeyPress(v *view.View, ev ui.KeyPress) {
	switch {
	case ev.Key == ui.KeyEnter:
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
	case ev.Key == ui.KeyBackspace:
		q0, q1 := v.Selected()
		if q0 == q1 {
			v.Select(q0-1, q1)
		}
		v.DeleteSel()
	case ev.Key == ui.KeyDelete:
		q0, q1 := v.Selected()
		if q0 == q1 {
			v.Select(q0, q1+1)
		}
		v.DeleteSel()
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
	(*view.View).ScrollUp(v, h)
}

func pageDown(v *view.View) {
	_, h := v.Size()
	(*view.View).ScrollDown(v, h)
}
