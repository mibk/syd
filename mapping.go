package main

import (
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

	parser.AddMovement(trans("j"), vi.DoN(down))
	parser.AddMovement(trans("k"), vi.DoN(up))
	parser.AddMovement(trans("h"), vi.DoN(left))
	parser.AddMovement(trans("l"), vi.DoN(right))

	parser.AddMovement(trans("G"), gotoLine)
	parser.AddAlias(trans("gg"), trans("1G"))
	parser.AddMovement(trans("|"), gotoColumn)
	parser.AddAlias(trans("0"), trans("|"))
	parser.AddMovement(trans("$"), doOnce(gotoEOL))

	parser.AddCommand([]event.KeyPress{{Key: 'f', Ctrl: true}}, vi.DoN(pageDown))
	parser.AddCommand([]event.KeyPress{{Key: 'b', Ctrl: true}}, vi.DoN(pageUp))

	parser.AddCommand(trans("i"), doOnce(insertMode))
	parser.AddCommand(trans(":"), doOnce(commandMode))
	parser.AddCommand(trans("u"), vi.DoN(undo))
	parser.AddCommand([]event.KeyPress{{Key: 'r', Ctrl: true}}, vi.DoN(redo))

	parser.AddCommand(trans("d"), doNAndCommit(delete), vi.RequiresMotion)
	parser.AddCommand(trans("x"), doNAndCommit(deleteRune))
	parser.AddAlias(trans("X"), trans("hx"))
}

func quit()        { shouldQuit = true }
func saveAndQuit() { checkAndSave(); quit() }

func down()  { viewport.GotoLine(viewport.Line() + 1); linewise() }
func up()    { viewport.GotoLine(viewport.Line() - 1); linewise() }
func right() { viewport.GotoColumn(viewport.Column() + 1); charwise() }
func left()  { viewport.GotoColumn(viewport.Column() - 1); charwise() }

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
}

func deleteRune() {
	c := viewport.CurrentCell()
	l := utf8.RuneLen(c.Rune)
	textBuf.Delete(c.Offset, l)
}
