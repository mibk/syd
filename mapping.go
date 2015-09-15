package main

import (
	"github.com/mibk/syd/event"
	"github.com/mibk/syd/vi"
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

	// ugly hack for the moment
	parser.AddAlias(trans("gg"), trans("1000k"))
	parser.AddAlias(trans("G"), trans("1000j"))

	parser.AddCommand(trans("i"), doOnce(insertMode))
	parser.AddCommand(trans(":"), doOnce(commandMode))
	parser.AddCommand(trans("u"), vi.DoN(undo))
	parser.AddCommand([]event.KeyPress{{Key: 'r', Ctrl: true}}, vi.DoN(redo))
}

func quit()        { shouldQuit = true }
func saveAndQuit() { checkAndSave(); quit() }

func down()  { viewport.MoveDown() }
func up()    { viewport.MoveUp() }
func left()  { viewport.MoveLeft() }
func right() { viewport.MoveRight() }

func undo() { textBuf.Undo() }
func redo() { textBuf.Redo() }
