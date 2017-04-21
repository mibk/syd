package core

import (
	"io"
	"unicode/utf8"

	"github.com/mibk/syd/undo"
)

type Buffer interface {
	ReadRuneAt(q int64) (r rune, size int, err error)
	Insert(q int64, s string)
	Delete(q0, q1 int64)
}

type BasicBuffer struct {
	runes []rune
}

func (bb *BasicBuffer) ReadRuneAt(q int64) (r rune, size int, err error) {
	i := int(q)
	if i >= len(bb.runes) {
		return 0, 0, io.EOF
	}
	r = bb.runes[i]
	return r, utf8.RuneLen(r), nil
}

func (bb *BasicBuffer) Insert(q int64, s string) {
	bb.runes = append(bb.runes[:q], append([]rune(s), bb.runes[q:]...)...)
}

func (bb *BasicBuffer) Delete(q0, q1 int64) {
	bb.runes = append(bb.runes[:q0], bb.runes[q1:]...)
}

type UndoBuffer struct {
	buf    *undo.Buffer
	offset int64 // offset in bytes
	pos    int64 // position in runes

	rb [4]byte // rune buffer
}

func NewUndoBuffer(buf *undo.Buffer) *UndoBuffer {
	return &UndoBuffer{
		buf: buf,
	}
}

func (b *UndoBuffer) ReadRuneAt(pos int64) (r rune, size int, err error) {
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

func (b *UndoBuffer) setPos(pos int64) (offset int64) {
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

func (b *UndoBuffer) readRuneAtByteOffset(off int64) (rune, int, error) {
	n, err := b.buf.ReadAt(b.rb[:], off)
	if n == 0 && err != nil {
		return 0, 0, err
	}
	r, s := utf8.DecodeRune(b.rb[:n])
	return r, s, nil
}

func (b *UndoBuffer) Insert(q int64, s string) {
	b.setPos(q)
	b.buf.Insert(b.offset, []byte(s))
}

func (b *UndoBuffer) Delete(q0, q1 int64) {
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
