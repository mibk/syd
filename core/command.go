package core

import (
	"errors"
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
			win.body.Select(win.buf.Undo())
		case "Redo":
			win.body.Select(win.buf.Redo())
		}
	default:
		shellexec(ctx, command)
	}
}

type writeFlusher interface {
	io.Writer
	flush()
}

func shellexec(ctx cmdContext, command string) {
	ed := ctx.editor()

	var stdin io.Reader
	stderr := ed.stderr()
	defer stderr.flush()
	stdout := stderr

	pipeln, err := parse(command)
	if err != nil {
		if err != errEmptyCmd {
			fmt.Fprintf(stderr, "syntax error: %v\n", err)
		}
		return
	}

	if pipeln.pipeInput || pipeln.pipeOutput {
		win, ok := ctx.window()
		if !ok {
			fmt.Fprintln(stderr, "no current window")
			return
		}
		if pipeln.pipeInput {
			// TODO: Implement this using io.Reader; read directly
			// from the buffer.
			q0, q1 := win.body.Selected()
			selected := win.body.SelectionToString(q0, q1)
			stdin = strings.NewReader(selected)
		}
		if pipeln.pipeOutput {
			stdout = win
			defer stdout.flush()
		}
	}

	if err := pipeln.Exec(stdin, stdout, stderr); err != nil {
		fmt.Fprintln(stderr, err)
		return
	}
}

var errEmptyCmd = errors.New("empty command")

func parse(s string) (*pipeline, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, errEmptyCmd
	}
	p := &pipeline{}
	switch r := s[0]; r {
	case '<', '|', '>':
		s = s[1:]
		p.pipeInput = true
		p.pipeOutput = true
		if r == '<' {
			p.pipeInput = false
		} else if r == '>' {
			p.pipeOutput = false
		}
	}

	var err error
	p.pipe, err = parsePipe(s)
	return p, err
}

func parsePipe(s string) (*pipe, error) {
	p := &pipe{}

	var err error
	i := strings.LastIndexByte(s, '|')
	if i == -1 {
		p.cmd, err = parseCmd(s)
		return p, err
	}

	p.cmd, err = parseCmd(s[i+1:])
	if err != nil {
		return nil, err
	}
	p.prev, err = parsePipe(s[:i])
	if err != nil {
		return nil, err
	}
	return p, nil
}

func parseCmd(s string) (*command, error) {
	args := strings.Fields(s)
	if len(args) == 0 {
		return nil, errors.New("missing command")
	}
	return &command{cmd: args[0], args: args[1:]}, nil
}

type pipeline struct {
	pipeInput  bool
	pipeOutput bool
	*pipe
}

type pipe struct {
	cmd  *command
	prev *pipe
}

func (p *pipe) Exec(stdin io.Reader, stdout, stderr io.Writer) error {
	wc, err := p.cmd.Start(stdout, stderr)
	if err != nil {
		return err
	}

	if p.prev != nil {
		if err := p.prev.Exec(stdin, wc, stderr); err != nil {
			return err
		}
	} else if stdin != nil {
		if _, err := io.Copy(wc, stdin); err != nil {
			return err
		}
	}
	return wc.Close()
}

type command struct {
	cmd  string
	args []string
}

func (c *command) Start(stdout, stderr io.Writer) (io.WriteCloser, error) {
	cmd := exec.Command(c.cmd, c.args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	wc, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &procCloser{wc, cmd}, nil
}

type procCloser struct {
	io.WriteCloser
	cmd *exec.Cmd
}

func (pc *procCloser) Close() error {
	if err := pc.WriteCloser.Close(); err != nil {
		return err
	}
	return pc.cmd.Wait()
}
