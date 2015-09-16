package main

import (
	"unicode/utf8"

	"github.com/mibk/syd/event"
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
	parser.AddCommand(trans("$"), doOnce(gotoEOL))

	parser.AddCommand([]event.KeyPress{{Key: 'f', Ctrl: true}}, vi.DoN(pageDown))
	parser.AddCommand([]event.KeyPress{{Key: 'b', Ctrl: true}}, vi.DoN(pageUp))

	parser.AddCommand(trans("i"), doOnce(insertMode))
	parser.AddCommand(trans(":"), doOnce(commandMode))
	parser.AddCommand(trans("u"), vi.DoN(undo))
	parser.AddCommand([]event.KeyPress{{Key: 'r', Ctrl: true}}, vi.DoN(redo))

	parser.AddCommand(trans("x"), doNAndCommit(deleteRune))
	parser.AddAlias(trans("X"), trans("hx"))
}

func quit()        { shouldQuit = true }
func saveAndQuit() { checkAndSave(); quit() }

func down()  { viewport.GotoLine(viewport.Line() + 1) }
func up()    { viewport.GotoLine(viewport.Line() - 1) }
func right() { viewport.GotoColumn(viewport.Column() + 1) }
func left()  { viewport.GotoColumn(viewport.Column() - 1) }

func gotoLine(n int) {
	if n == 0 {
		n = view.Last
	} else {
		n--
	}
	viewport.GotoLine(n)
}
func gotoEOL()         { viewport.GotoColumn(view.Last) }
func gotoColumn(n int) { viewport.GotoColumn(n - 1) }

func pageDown() { viewport.SetFirstLine(viewport.FirstLine() + viewport.Height()) }
func pageUp()   { viewport.SetFirstLine(viewport.FirstLine() - viewport.Height()) }

func undo() { textBuf.Undo() }
func redo() { textBuf.Redo() }

func deleteRune() {
	c := viewport.CurrentCell()
	l := utf8.RuneLen(c.Rune)
	textBuf.Delete(c.Offset, l)
}
