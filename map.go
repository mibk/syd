package main

import (
	"github.com/mibk/syd/ui"
	"github.com/mibk/syd/view"
)

func setMappings(syd *Syd) {
	syd.AddStringOperator("q", doOnce(func(*view.View) { syd.shouldQuit = true }))

	syd.AddStringMotion("j", doNTimes(down))
	syd.AddStringMotion("k", doNTimes(up))
	syd.AddStringMotion("h", doNTimes(left))
	syd.AddStringMotion("l", doNTimes(right))

	syd.AddOperator([]ui.KeyPress{{Key: 'f', Ctrl: true}}, doNTimes(pageDown))
	syd.AddOperator([]ui.KeyPress{{Key: 'b', Ctrl: true}}, doNTimes(pageUp))

	syd.AddStringOperator("d", doNTimes(func(v *view.View) { v.DeleteSel() }))

	syd.AddStringOperator("u", doNTimes((*view.View).Undo))
	syd.AddOperator([]ui.KeyPress{{Key: 'r', Ctrl: true}}, doNTimes((*view.View).Redo))

	syd.AddStringOperator("i", doOnce(func(*view.View) { syd.mode = ModeInsert }))
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
