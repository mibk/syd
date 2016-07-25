package main

import (
	"github.com/mibk/syd/ui"
	"github.com/mibk/syd/view"
)

func mapCommands(syd *Syd) {
	syd.AddStringCommand("q", doOnce(func(*view.View) { syd.shouldQuit = true }))

	syd.AddStringMotion("j", doNTimes(down))
	syd.AddStringMotion("k", doNTimes(up))
	syd.AddStringMotion("h", doNTimes(left))
	syd.AddStringMotion("l", doNTimes(right))

	syd.AddCommand([]ui.KeyPress{{Key: 'f', Ctrl: true}}, doNTimes(pageDown))
	syd.AddCommand([]ui.KeyPress{{Key: 'b', Ctrl: true}}, doNTimes(pageUp))

	syd.AddStringCommand("v", doNTimes(visualMode))

	syd.AddStringCommand("d", doNTimes(func(v *view.View) {
		v.DelSelected()
		qvis = -1
	}))

	syd.AddStringCommand("u", doNTimes((*view.View).Undo))
	syd.AddCommand([]ui.KeyPress{{Key: 'r', Ctrl: true}}, doNTimes((*view.View).Redo))

	syd.AddStringCommand("i", doOnce(func(*view.View) { syd.mode = ModeInsert }))
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
