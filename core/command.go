package core

import (
	"fmt"
	"io"
	"os/exec"
	"strings"

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
		}
	default:
		shell(ctx, command)
	}
}

type writeFlusher interface {
	io.Writer
	flush()
}

func shell(ctx cmdContext, command string) {
	ed := ctx.editor()

	var stdin io.Reader
	stderr := ed.stderr()
	defer stderr.flush()
	stdout := stderr

	if command[0] == '|' {
		win, ok := ctx.window()
		if !ok {
			fmt.Fprintln(stderr, "no current window")
			return
		}
		command = command[1:]

		// TODO: Implement this using io.Reader; read directly
		// from the buffer.
		q0, q1 := win.body.Selected()
		selected := win.body.SelectionToString(q0, q1)
		stdin = strings.NewReader(selected)
		stdout = win
	}

	cmd := exec.Command(command)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintln(stderr, err)
	}

	stdout.flush()
}
