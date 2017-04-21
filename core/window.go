package core

import (
	"os"
	"unicode/utf8"

	"github.com/mibk/syd/ui/term"
)

const EOF = utf8.MaxRune + 1

type Window struct {
	col      *Column
	filename string
	win      *term.Window
	con      Content

	buf  *UndoBuffer
	tag  *Text
	body *Text
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

func (win *Window) Undo() { win.buf.Undo() }
func (win *Window) Redo() { win.buf.Redo() }

func (win *Window) Close() error {
	win.win.Delete()
	win.col.deleteWindow(win)
	return win.con.Close()
}

func (win *Window) saveFile() {
	if win.filename == "" {
		win.readFilename()
	}
	// TODO: Read bytes directly from the undo.Buffer.
	// TODO: Don't use '~' suffix, make saving safer.
	f, err := os.Create(win.filename + "~")
	if err != nil {
		panic(err)
	}

	var buf [64]byte
	var i int

	for p := int64(0); ; p++ {
		r := win.body.ReadRuneAt(p)
		if r == EOF || len(buf[i:]) < utf8.UTFMax {
			if _, err := f.Write(buf[:i]); err != nil {
				panic(err)
			}
			i = 0
		}
		if r == EOF {
			break
		}
		i += utf8.EncodeRune(buf[i:], r)
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

func (win *Window) editor() (ed *Editor)           { return win.col.ed }
func (win *Window) column() (col *Column, ok bool) { return win.col, true }
func (win *Window) window() (w *Window, ok bool)   { return win, true }
