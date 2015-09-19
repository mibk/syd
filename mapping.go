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
	parser.AddMotion(trans("^"), doOnce(gotoFirstNonBlank))
	parser.AddMotion(trans("$"), doOnce(gotoEOL))
	parser.AddMotion(trans("_"), underscore)

	parser.AddMotion(trans("H"), gotoScreenLineFromTop)
	parser.AddMotion(trans("L"), gotoScreenLineFromBotton)
	parser.AddMotion(trans("M"), doOnce(gotoMiddleScreenLine))
	parser.AddCommand(trans("zt"), doOnce(setScreenLineTop))
	parser.AddCommand(trans("zz"), doOnce(setScreenLineMiddle))
	parser.AddCommand(trans("zb"), doOnce(SetScreenLineBottom))

	parser.AddCommand([]event.KeyPress{{Key: 'f', Ctrl: true}}, vi.DoN(pageDown))
	parser.AddCommand([]event.KeyPress{{Key: 'b', Ctrl: true}}, vi.DoN(pageUp))

	parser.AddCommand(trans("i"), doOnce(insertMode))
	parser.AddCommand(trans("a"), doOnce(appendRight))
	parser.AddCommand(trans("o"), doOnce(openLineDown))
	parser.AddCommand(trans("O"), doOnce(openLineUp))
	parser.AddAlias(trans("I"), trans("^i"))
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

	parser.AddCommand(trans("y"), doOnce(yank), vi.RequiresMotion)
	parser.AddAlias(trans("yy"), trans("y_"))
	parser.AddAlias(trans("Y"), trans("y$"))
	parser.AddCommand(trans("P"), doOnce(Paste))
	parser.AddCommand(trans("p"), doOnce(paste))
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
func gotoFirstNonBlank() {
	off := viewport.CurrentCell().Offset
	start := textutil.FindLineStart(textBuf, int64(off))
	off = int(textutil.FindIndentOffset(textBuf, start))
	viewport.SetCursor(off)
	charwise()
}

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

func setScreenLineTop()    { setScreenLinePos(0) }
func setScreenLineMiddle() { setScreenLinePos(viewport.Height()/2 - 1) }
func SetScreenLineBottom() { setScreenLinePos(viewport.Height() - 1) }
func setScreenLinePos(pos int) {
	s := viewport.ScreenLine()
	f := viewport.FirstLine()
	l := viewport.Line()
	viewport.SetFirstLine(f - (pos - s))
	viewport.GotoLine(l)
}

func pageDown() { viewport.SetFirstLine(viewport.FirstLine() + viewport.Height()) }
func pageUp()   { viewport.SetFirstLine(viewport.FirstLine() - viewport.Height()) }

func undo() { textBuf.Undo() }
func redo() { textBuf.Redo() }

func delete() {
	start, end, desiredOffset := findBorders()
	textBuf.Delete(start, end-start)
	viewport.SetCursor(desiredOffset)
	lastOffset = start
}

func findBorders() (off1, off2, desiredOffset int) {
	off1 = lastOffset
	off2 = viewport.CurrentCell().Offset
	if off1 > off2 {
		off1, off2 = off2, off1
	}
	desiredOffset = off1
	if isLinewise {
		off1 = int(textutil.FindLineStart(textBuf, int64(off1)))
		off2 = int(textutil.FindLineEnd(textBuf, int64(off2)))
	}
	return
}

func yank() {
	start, end, desiredOffset := findBorders()

	clipboard = make([]byte, end-start)
	textBuf.ReadAt(clipboard, int64(start))
	wasCopiedLinewise = isLinewise

	viewport.SetCursor(desiredOffset)
	lastOffset = start
}
func paste() {
	if clipboard == nil {
		return
	}
	off := lastOffset
	if wasCopiedLinewise {
		off = int(textutil.FindLineEnd(textBuf, int64(off)))
		down()
	} else {
		right()
	}
	textBuf.Insert(off, clipboard)
}
func Paste() {
	if clipboard == nil {
		return
	}
	off := lastOffset
	if wasCopiedLinewise {
		off = int(textutil.FindLineStart(textBuf, int64(off)))
	}
	textBuf.Insert(off, clipboard)
}

func appendRight() {
	right()
	insertMode()
}

func openLineDown() {
	end := int(textutil.FindLineEnd(textBuf, int64(lastOffset)))
	start := int(textutil.FindLineStart(textBuf, int64(lastOffset)))
	openLine(end, start)
}

func openLineUp() {
	off := int(textutil.FindLineStart(textBuf, int64(lastOffset)))
	openLine(off, off)
}

func openLine(off, start int) {
	ioffset := textutil.FindIndentOffset(textBuf, int64(start))
	b := make([]byte, int(ioffset)-start+1)
	textBuf.ReadAt(b[:len(b)-1], int64(start))
	b[len(b)-1] = '\n'
	textBuf.Insert(off, b)
	viewport.SetCursor(off + len(b) - 1)
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
