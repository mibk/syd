package core

import (
	"bytes"
	"io"
	"os/exec"
	"strings"
	"unicode/utf8"

	"github.com/mibk/syd/ui"
)

type cmdContext interface {
	editor() (ed *Editor)
	column() (col *Column, ok bool)
	window() (win *Window, ok bool)
}

func execute(ctx cmdContext, command string) {
	if command == "" {
		return
	}
	// TODO: Print err if the context isn't sufficient.
	switch command {
	case "Exit":
		// TODO: This is just a temporary solution
		// until a proper solution is found.
		go func() {
			ui.Events <- ui.Quit
		}()

	case "Newcol":
		ctx.editor().NewColumn()

	case "Delcol", "New":
		col, ok := ctx.column()
		if !ok {
			return
		}
		switch command {
		case "Delcol":
			col.Close()
		case "New":
			col.NewWindow()
		}

	case "Del", "Put", "Undo", "Redo":
		fallthrough
	default:
		win, ok := ctx.window()
		if !ok {
			return
		}
		switch command {
		case "Del":
			win.Close()
		case "Put":
			win.saveFile()
			win.buf.Clean()
		case "Undo":
			win.buf.Undo()
		case "Redo":
			win.buf.Redo()
		default:
			ed := win.col.ed
			if ed.errWin == nil {
				ed.errWin = ed.recentCol().NewWindow()
				ed.errWin.SetFilename("+Errors")
				// TODO: This is just a hack because one
				// cannot write to a window until this method
				// is at least once called. Remove it.
				ed.errWin.win.Clear()
			}
			wout := ed.errWin

			var stdin io.Reader
			if command[0] == '|' {
				command = command[1:]

				// TODO: Implement this using io.Reader; read directly
				// from the buffer.
				q0, q1 := win.body.Selected()
				selected := win.body.SelectionToString(q0, q1)
				stdin = strings.NewReader(selected)
				wout = win
			} else {
				q := wout.body.buf.End()
				wout.body.q0, wout.body.q1 = q, q
			}

			var buf bytes.Buffer
			cmd := exec.Command(command)
			cmd.Stdin = stdin
			cmd.Stdout = &buf
			// TODO: Redirect stderr somewhere.
			switch err := cmd.Run(); err := err.(type) {
			case *exec.Error:
				if err.Err == exec.ErrNotFound {
					return
				}
				panic(err)
			case error:
				panic(err)
			}
			s := buf.String()
			q := wout.body.q0
			wout.body.Insert(s)
			wout.body.Select(q, q+int64(utf8.RuneCountInString(s)))

			// TODO: Come up with a better solution
			wout.buf.Commit()
		}
	}
}
