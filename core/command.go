package core

import (
	"bytes"
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
			if win.filename == "" {
				var runes []rune
				var p int64
				for {
					r := win.tag.ReadRuneAt(p)
					if r == 0 || r == EOF {
						break
					}
					runes = append(runes, r)
					p++
				}
				if len(runes) == 0 {
					return
				}
				win.filename = string(runes)
			}
			if err := win.saveFile(); err != nil {
				panic(err)
			}
			win.buf.Clean()
		case "Undo":
			win.Undo()
		case "Redo":
			win.Redo()
		default:
			if command[0] != '|' {
				return
			}
			command = command[1:]

			// TODO: Implement this using io.Reader; read directly
			// from the buffer.
			q0, q1 := win.body.Selected()
			selected := win.body.SelectionToString(q0, q1)
			var buf bytes.Buffer
			rd := strings.NewReader(selected)
			cmd := exec.Command(command)
			cmd.Stdin = rd
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
			win.body.Insert(s)
			win.body.Select(q0, q0+int64(utf8.RuneCountInString(s)))

			// TODO: Come up with a better solution
			win.buf.buf.Commit()
		}
	}
}
