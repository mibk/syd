package main

import "github.com/mibk/syd/vi"

func doOnce(f func()) func(int) {
	return func(_ int) {
		f()
	}
}

func performMapping() {
	parser.AddCommand("ZQ", doOnce(quit))
	parser.AddCommand("ZZ", doOnce(saveAndQuit))

	parser.AddMovement("j", vi.DoN(down))
	parser.AddMovement("k", vi.DoN(up))
	parser.AddMovement("h", vi.DoN(left))
	parser.AddMovement("l", vi.DoN(right))

	// ugly hack for the moment
	parser.AddAlias("gg", "1000k")
	parser.AddAlias("G", "1000j")

	parser.AddCommand("i", doOnce(insertMode))
	parser.AddCommand(":", doOnce(commandMode))
	parser.AddCommand("u", vi.DoN(undo))
}

func quit()        { shouldQuit = true }
func saveAndQuit() { checkAndSave(); quit() }

func down()  { viewport.MoveDown() }
func up()    { viewport.MoveUp() }
func left()  { viewport.MoveLeft() }
func right() { viewport.MoveRight() }

func undo() { textBuf.Undo() }
