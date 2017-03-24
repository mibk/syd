package main

import (
	"github.com/mibk/syd/ui"
	"github.com/mibk/syd/view"
)

func setMappings(e *Editor) {
	e.AddStringOperator("q", doOnce(func(*view.View) { e.shouldQuit = true }))

	e.AddStringMotion("j", doNTimes(down))
	e.AddStringMotion("k", doNTimes(up))
	e.AddStringMotion("h", doNTimes(left))
	e.AddStringMotion("l", doNTimes(right))

	e.AddOperator([]ui.KeyPress{{Key: 'f', Ctrl: true}}, doNTimes(pageDown))
	e.AddOperator([]ui.KeyPress{{Key: 'b', Ctrl: true}}, doNTimes(pageUp))

	e.AddStringOperator("d", doNTimes(func(v *view.View) { v.DeleteSel() }))

	e.AddStringOperator("u", doNTimes((*view.View).Undo))
	e.AddOperator([]ui.KeyPress{{Key: 'r', Ctrl: true}}, doNTimes((*view.View).Redo))

	e.AddStringOperator("i", doOnce(func(*view.View) { e.mode = ModeInsert }))
}

func doOnce(fn func(*view.View)) func(*view.View, int) {
	return func(v *view.View, _ int) { fn(v) }
}

func doNTimes(fn func(*view.View)) func(*view.View, int) {
	return func(v *view.View, n int) {
		if n == 0 {
			n = 1
		}
		for ; n > 0; n-- {
			fn(v)
		}
	}
}
