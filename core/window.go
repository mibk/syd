package core

import (
	"bytes"
	"io"
	"os"
	"unicode/utf8"

	"github.com/mibk/syd/ui/term"
	"github.com/mibk/syd/undo"
)

const EOF = utf8.MaxRune + 1

type Window struct {
	col      *Column
	filename string
	win      *term.Window
	con      Content

	buf  *undo.Buffer
	tag  *Text
	body *Text

	// used by Read and flush methods
	insertbuf bytes.Buffer
}

func (win *Window) SetFilename(filename string) {
	win.filename = filename
	win.tag.buf.Insert(0, filename)
	win.col.ed.wins[filename] = win
}

func (win *Window) Frame() *term.Frame { return win.body.text.Frame() } // TODO: delete

func (win *Window) LoadText() {
	win.win.Clear()
	win.win.SetDirty(win.buf.Dirty())
	win.tag.loadText()
	win.body.loadText()
}

func (win *Window) Close() error {
	win.win.Delete()
	win.col.deleteWindow(win)
	if ed := win.col.ed; ed.errWin == win {
		ed.errWin = nil
	}
	return win.con.Close()
}

func (win *Window) Write(b []byte) (n int, err error) {
	return win.insertbuf.Write(b)
}

func (win *Window) flush() {
	s := win.insertbuf.String()
	win.insertbuf.Reset()
	q := win.body.q0
	win.body.Insert(s)
	win.body.Select(q, q+int64(utf8.RuneCountInString(s)))

	// TODO: Come up with a better solution?
	win.buf.Commit()
}

const maxInt64 = 1<<63 - 1

func (win *Window) saveFile() {
	if win.filename == "" {
		win.readFilename()
	}

	// TODO: Don't use '~' suffix, make saving safer.
	f, err := os.Create(win.filename + "~")
	if err != nil {
		panic(err)
	}
	r := io.NewSectionReader(win.buf, 0, maxInt64)
	if _, err := io.Copy(f, r); err != nil {
		panic(err)
	}
	f.Close()

	if err := os.Rename(win.filename+"~", win.filename); err != nil {
		panic(err)
	}
}

func (win *Window) readFilename() {
	var runes []rune
	var p int64
	for {
		r := win.tag.readRuneAt(p)
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

func (win *Window) editor() (ed *Editor)           { return win.col.ed }
func (win *Window) column() (col *Column, ok bool) { return win.col, true }
func (win *Window) window() (w *Window, ok bool)   { return win, true }
