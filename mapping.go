package main

import (
	"bytes"
	"unicode/utf8"

	"github.com/mibk/syd/event"
	"github.com/mibk/syd/textutil"
	"github.com/mibk/syd/vi"
	"github.com/mibk/syd/view"
)

func doOnce(f func()) func(int) {
	return func(_ int) {
		f()
	}
}

func doNAndCommit(f func()) func(int) {
	return func(n int) {
		vi.DoN(f)(n)
		textBuf.CommitChanges()
	}
}

func trans(cmd string) []event.KeyPress {
	events := make([]event.KeyPress, 0, len(cmd))
	for _, r := range []rune(cmd) {
		events = append(events, event.KeyPress{Key: event.Key(r)})
	}
	return events
}

func performMapping() {
	parser.AddCommand(trans("ZQ"), doOnce(quit))
	parser.AddCommand(trans("ZZ"), doOnce(saveAndQuit))

	parser.AddCommand(trans("."), vi.DoN(repeatLastAction))

	parser.AddMotion(trans("j"), vi.DoN(down))
	parser.AddMotion(trans("k"), vi.DoN(up))
	parser.AddMotion(trans("h"), vi.DoN(left))
	parser.AddMotion(trans("l"), vi.DoN(right))

	parser.AddMotion(trans("G"), gotoLine)
	parser.AddAlias(trans("gg"), trans("1G"))
	parser.AddMotion(trans("|"), gotoColumn)
	parser.AddAlias(trans("0"), trans("|"))
	parser.AddMotion(trans("$"), doOnce(gotoEOL))
	parser.AddMotion(trans("_"), underscore)

	parser.AddMotion(trans("H"), gotoScreenLineFromTop)
	parser.AddMotion(trans("L"), gotoScreenLineFromBotton)
	parser.AddMotion(trans("M"), doOnce(gotoMiddleScreenLine))

	parser.AddCommand([]event.KeyPress{{Key: 'f', Ctrl: true}}, vi.DoN(pageDown))
	parser.AddCommand([]event.KeyPress{{Key: 'b', Ctrl: true}}, vi.DoN(pageUp))

	parser.AddCommand(trans("i"), doOnce(insertMode))
	parser.AddCommand(trans("a"), doOnce(appendRight))
	parser.AddCommand(trans("o"), doOnce(openLineDown))
	parser.AddCommand(trans("O"), doOnce(openLineUp))
	parser.AddAlias(trans("I"), trans("|i"))
	parser.AddAlias(trans("A"), trans("$a"))

	parser.AddCommand(trans(":"), doOnce(commandMode))
	parser.AddCommand(trans("u"), vi.DoN(undo))
	parser.AddCommand([]event.KeyPress{{Key: 'r', Ctrl: true}}, vi.DoN(redo))

	parser.AddCommand(trans("d"), doNAndCommit(delete), vi.RequiresMotion)
	parser.AddAlias(trans("dd"), trans("d_"))
	parser.AddAlias(trans("D"), trans("d$"))
	parser.AddAlias(trans("x"), trans("dl"))
	parser.AddAlias(trans("X"), trans("dh"))

	parser.AddCommand(trans("c"), doNAndCommit(change), vi.RequiresMotion)
	parser.AddAlias(trans("cc"), trans("c_"))
	parser.AddAlias(trans("C"), trans("c$"))
	parser.AddAlias(trans("s"), trans("dli"))
	parser.AddAlias(trans("S"), trans("c_"))

	parser.AddCommand(trans("r"), replace)
}

func quit()        { shouldQuit = true }
func saveAndQuit() { checkAndSave(); quit() }

func repeatLastAction() {
	doNotRemember()
	lastAction()
}

func down()  { viewport.GotoLine(viewport.Line() + 1); linewise() }
func up()    { viewport.GotoLine(viewport.Line() - 1); linewise() }
func right() { viewport.GotoColumn(viewport.Column() + 1); charwise() }
func left()  { viewport.GotoColumn(viewport.Column() - 1); charwise() }

func underscore(n int) {
	if n != 0 {
		n--
	}
	viewport.GotoLine(viewport.Line() + n)
	linewise()
}

func gotoLine(n int) {
	if n == 0 {
		n = view.Last
	} else {
		n--
	}
	viewport.GotoLine(n)
	linewise()
}
func gotoEOL()         { viewport.GotoColumn(view.Last); charwise() }
func gotoColumn(n int) { viewport.GotoColumn(n - 1); charwise() }

func gotoScreenLineFromTop(n int) {
	if n != 0 {
		n--
	}
	viewport.GotoLine(viewport.Line() - viewport.ScreenLine() + n)
}
func gotoScreenLineFromBotton(n int) {
	if n != 0 {
		n--
	}
	viewport.GotoLine(viewport.Line() - viewport.ScreenLine() +
		viewport.Height() - n - 1)
}
func gotoMiddleScreenLine() { gotoScreenLineFromTop(viewport.Height() / 2) }

func pageDown() { viewport.SetFirstLine(viewport.FirstLine() + viewport.Height()) }
func pageUp()   { viewport.SetFirstLine(viewport.FirstLine() - viewport.Height()) }

func undo() { textBuf.Undo() }
func redo() { textBuf.Redo() }

func delete() {
	off1 := lastOffset
	off2 := viewport.CurrentCell().Offset
	if off1 > off2 {
		off1, off2 = off2, off1
	}
	desiredOffset := off1
	if isLinewise {
		off1 = int(textutil.FindLineStart(textBuf, int64(off1)))
		off2 = int(textutil.FindLineEnd(textBuf, int64(off2)))
	}
	textBuf.Delete(off1, off2-off1)
	viewport.SetCursor(desiredOffset)
	lastOffset = off1
}

func appendRight() {
	right()
	insertMode()
}

func openLineDown() {
	openLine(int(textutil.FindLineEnd(textBuf, int64(lastOffset))))
}

func openLineUp() {
	openLine(int(textutil.FindLineStart(textBuf, int64(lastOffset))))
}

func openLine(off int) {
	textBuf.Insert(off, []byte{'\n'})
	viewport.SetCursor(off)
	insertMode()
}

func change() {
	delete()
	if isLinewise {
		openLineUp()
	} else {
		insertMode()
	}
}

func replace(n int) {
	if n == 0 {
		n = 1
	}
	ev := event.PollEvent()
	if ev, ok := ev.(event.KeyPress); ok {
		viewport.GotoColumn(viewport.Column() + n)
		delete()
		p := make([]byte, 4)
		length := utf8.EncodeRune(p, rune(ev.Key))
		p = bytes.Repeat(p[:length], n)
		textBuf.Insert(lastOffset, p)
	}
}
