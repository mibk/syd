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
	End() (q int64)
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

func (bb *BasicBuffer) End() int64 { return int64(len(bb.runes)) }

type UndoBuffer struct {
	*undo.Buffer
	offset int64 // offset in bytes
	pos    int64 // position in runes

	rb [4]byte // rune buffer
}

func NewUndoBuffer(buf *undo.Buffer) *UndoBuffer {
	return &UndoBuffer{
		Buffer: buf,
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

// RuneReaderFrom returns an io.RuneReader and the offset in bytes
// that corresponds to q.
func (b *UndoBuffer) RuneReaderFrom(q int64) (r io.RuneReader, off int64) {
	off = b.setPos(q)
	return &posRuneReader{b: b, q: q}, off
}

func (b *UndoBuffer) Insert(q int64, s string) {
	b.setPos(q)
	b.Buffer.Insert(b.offset, []byte(s))
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
	if err := b.Buffer.Delete(offset, size); err != nil {
		panic(err)
	}
}

func (b *UndoBuffer) Undo() (q0, q1 int64) { return b.FindRange(b.Buffer.Undo()) }
func (b *UndoBuffer) Redo() (q0, q1 int64) { return b.FindRange(b.Buffer.Redo()) }

func (b *UndoBuffer) FindRange(off, n int64) (q0, q1 int64) {
	if off == -1 {
		return -1, -1
	}
	q0 = b.setOffset(off)
	q1 = b.setOffset(off + n)
	return
}

func (b *UndoBuffer) End() int64 {
	p := b.pos
	for ; ; p++ {
		if _, _, err := b.ReadRuneAt(p); err != nil {
			break
		}
	}
	return p
}

func (b *UndoBuffer) setPos(pos int64) (offset int64) {
	if pos < b.pos {
		b.offset = 0
		b.pos = 0
	}
	for {
		if b.pos == pos {
			return b.offset
		}
		b.advancePos()
	}
}

func (b *UndoBuffer) setOffset(off int64) (pos int64) {
	if off < b.offset {
		b.offset = 0
		b.pos = 0
	}
	for {
		if b.offset >= off {
			return b.pos
		}
		b.advancePos()
	}
}

func (b *UndoBuffer) advancePos() {
	_, size, err := b.readRuneAtByteOffset(b.offset)
	if err != nil {
		panic(err)
	}
	b.offset += int64(size)
	b.pos++
}

func (b *UndoBuffer) readRuneAtByteOffset(off int64) (rune, int, error) {
	n, err := b.Buffer.ReadAt(b.rb[:], off)
	if n == 0 && err != nil {
		return 0, 0, err
	}
	r, s := utf8.DecodeRune(b.rb[:n])
	return r, s, nil
}

type posRuneReader struct {
	b *UndoBuffer
	q int64
}

func (rr *posRuneReader) ReadRune() (r rune, size int, err error) {
	r, size, err = rr.b.ReadRuneAt(rr.q)
	rr.q++
	return
}
