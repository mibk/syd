package core

import (
	"io"
	"unicode/utf8"

	"github.com/mibk/syd/libs/undo"
)

type Buffer struct {
	buf    *undo.Buffer
	offset int64 // offset in bytes
	pos    int64 // position in runes

	rb []byte // rune buffer
}

func NewBuffer(buf *undo.Buffer) *Buffer {
	return &Buffer{
		buf: buf,
		rb:  make([]byte, 4),
	}
}

func (b *Buffer) ReadRuneAt(pos int64) (r rune, size int, err error) {
	if pos < b.pos {
		b.offset = 0
		b.pos = 0
	}
	for {
		r, s, err := b.readRuneAtByteOffset(b.offset)
		if err != nil {
			return 0, 0, err
		}
		b.offset += int64(s)
		b.pos++
		if pos == b.pos-1 {
			return r, s, nil
		}
	}
}

func (b *Buffer) setPos(pos int64) (offset int64) {
	if pos < b.pos {
		b.offset = 0
		b.pos = 0
	}
	for {
		if pos == b.pos {
			return b.offset
		}
		_, s, err := b.readRuneAtByteOffset(b.offset)
		if err != nil {
			panic(err)
		}
		b.offset += int64(s)
		b.pos++
	}
}

func (b *Buffer) readRuneAtByteOffset(off int64) (rune, int, error) {
	n, err := b.buf.ReadAt(b.rb, off)
	if n == 0 && err != nil {
		return 0, 0, err
	}
	r, s := utf8.DecodeRune(b.rb)
	return r, s, nil
}

func (b *Buffer) Insert(q int64, s string) {
	b.setPos(q)
	b.buf.Insert(b.offset, []byte(s))
}

func (b *Buffer) Delete(q0, q1 int64) {
	var size int64
	offset := b.setPos(q0)
	for l := q1 - q0; l > 0; l-- {
		_, s, err := b.ReadRuneAt(q0)
		if err == io.EOF {
			return
		} else if err != nil {
			panic(err)
		}
		size += int64(s)
		q0++
	}
	if err := b.buf.Delete(offset, size); err != nil {
		panic(err)
	}
}

func (b *Buffer) Undo() { b.buf.Undo() }
func (b *Buffer) Redo() { b.buf.Redo() }
