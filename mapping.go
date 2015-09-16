package main

import (
	"github.com/mibk/syd/event"
	"github.com/mibk/syd/vi"
	"github.com/mibk/syd/view"
)

func doOnce(f func()) func(int) {
	return func(_ int) {
		f()
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

	parser.AddCommand(trans("i"), doOnce(insertMode))
	parser.AddCommand(trans(":"), doOnce(commandMode))
	parser.AddCommand(trans("u"), vi.DoN(undo))
	parser.AddCommand([]event.KeyPress{{Key: 'r', Ctrl: true}}, vi.DoN(redo))
}

func quit()        { shouldQuit = true }
func saveAndQuit() { checkAndSave(); quit() }

func down()  { viewport.GotoLine(viewport.Line() + 1) }
func up()    { viewport.GotoLine(viewport.Line() - 1) }
func left()  { viewport.MoveLeft() }
func right() { viewport.MoveRight() }

func gotoLine(n int) {
	if n == 0 {
		n = view.LastLine
	} else {
		n--
	}
	viewport.GotoLine(n)
}

func undo() { textBuf.Undo() }
func redo() { textBuf.Redo() }
