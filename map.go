package main

import (
	"github.com/mibk/syd/ui"
	"github.com/mibk/syd/view"
)

func setMappings(ed *Editor) {
	ed.AddStringOperator("q", doOnce(func(*view.View) { ed.shouldQuit = true }))

	ed.AddStringMotion("j", doNTimes(down))
	ed.AddStringMotion("k", doNTimes(up))
	ed.AddStringMotion("h", doNTimes(left))
	ed.AddStringMotion("l", doNTimes(right))

	ed.AddOperator([]ui.KeyPress{{Key: 'f', Ctrl: true}}, doNTimes(pageDown))
	ed.AddOperator([]ui.KeyPress{{Key: 'b', Ctrl: true}}, doNTimes(pageUp))

	ed.AddStringOperator("d", doNTimes(func(v *view.View) { v.DeleteSel() }))

	ed.AddStringOperator("u", doNTimes((*view.View).Undo))
	ed.AddOperator([]ui.KeyPress{{Key: 'r', Ctrl: true}}, doNTimes((*view.View).Redo))

	ed.AddStringOperator("i", doOnce(func(*view.View) { ed.mode = ModeInsert }))
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
