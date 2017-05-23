package core

import (
	"bytes"
	"io"
	"os"
	"regexp"
	"unicode/utf8"

	"github.com/mibk/syd/ui"
)

const EOF = utf8.MaxRune + 1

type Window struct {
	col      *Column
	filename string
	win      ui.Updater
	con      Content

	buf  *UndoBuffer
	tag  *Text
	body *Text

	// used by Read and flush methods
	insertbuf bytes.Buffer

	y float64
}

func (win *Window) SetFilename(filename string) {
	win.filename = filename
	win.tag.buf.Insert(0, filename)
	win.col.ed.wins[filename] = win
}

func (win *Window) Dirty() bool {
	return win.buf.Dirty()
}

func (win *Window) MoveToColumn(col *Column) {
	if win.col == col {
		return
	}
	win.col.deleteWindow(win)
	win.col = col
	col.wins = append(col.wins, win)
}

func (win *Window) Y() float64 { return win.y }

func (win *Window) SetY(y float64) {
	if y < 0 || y > 1 {
		panic("y must be in the range 0..1")
	}
	win.y = y
}

func (win *Window) Close() error {
	win.win.Update(ui.Delete)
	win.col.deleteWindow(win)
	if ed := win.col.ed; ed.errWin == win {
		ed.errWin = nil
	}
	if win.filename != "" {
		delete(win.col.ed.wins, win.filename)
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

func (win *Window) saveFile() {
	if win.filename == "" {
		win.readFilename()
	}

	// TODO: Don't use '~' suffix, make saving safer.
	f, err := os.Create(win.filename + "~")
	if err != nil {
		panic(err)
	}
	r := io.NewSectionReader(win.buf, 0, win.buf.Size())
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

func (win *Window) findNextExactMatch(s string) {
	rx := regexp.MustCompile(regexp.QuoteMeta(s))

	body := win.body
	buf := win.buf
	for _, q := range []int64{body.q1, 0} {
		r, off := buf.RuneReaderFrom(q)
		if loc := rx.FindReaderIndex(r); loc != nil {
			q0, q1 := buf.FindRange(off+int64(loc[0]), int64(loc[1]-loc[0]))
			body.Select(q0, q1)
			return
		}
	}
}

func (win *Window) editor() (ed *Editor)           { return win.col.ed }
func (win *Window) column() (col *Column, ok bool) { return win.col, true }
func (win *Window) window() (w *Window, ok bool)   { return win, true }
