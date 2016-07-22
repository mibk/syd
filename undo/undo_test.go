package undo

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

func TestOverall(t *testing.T) {
	b := NewBuffer(nil)
	b.checkPiecesCnt(t, 2)
	b.checkContent("#0", t, "")

	b.insertString(0, "")
	b.checkPiecesCnt(t, 2)
	b.checkContent("#1", t, "")

	b.insertString(0, "All work makes John a dull boy")
	b.checkPiecesCnt(t, 3)
	b.checkContent("#2", t, "All work makes John a dull boy")

	b.insertString(9, "and no playing ")
	b.checkPiecesCnt(t, 6)
	b.checkContent("#3", t, "All work and no playing makes John a dull boy")

	b.CommitChanges()
	// Also check that multiple change commits don't create empty changes.
	b.CommitChanges()
	b.Delete(20, 14)
	b.checkContent("#4", t, "All work and no play a dull boy")

	b.insertString(20, " makes Jack")
	b.checkContent("#5", t, "All work and no play makes Jack a dull boy")

	b.Undo()
	b.checkContent("#6", t, "All work and no play a dull boy")
	b.Undo()
	b.checkContent("#7", t, "All work and no playing makes John a dull boy")
	b.Undo()
	b.checkContent("#8", t, "All work makes John a dull boy")

	b.Redo()
	b.checkContent("#9", t, "All work and no playing makes John a dull boy")
	b.Redo()
	b.checkContent("#10", t, "All work and no play a dull boy")
	b.Redo()
	b.checkContent("#11", t, "All work and no play makes Jack a dull boy")
	b.Redo()
	b.checkContent("#12", t, "All work and no play makes Jack a dull boy")
}

func TestCacheInsertAndDelete(t *testing.T) {
	b := NewBuffer([]byte("testing insertation"))
	b.checkPiecesCnt(t, 3)
	b.checkContent("#0", t, "testing insertation")

	b.cacheInsertString(8, "caching")
	b.checkPiecesCnt(t, 6)
	b.checkContent("#1", t, "testing cachinginsertation")

	b.cacheInsertString(15, " ")
	b.checkPiecesCnt(t, 6)
	b.checkContent("#2", t, "testing caching insertation")

	b.cacheDelete(12, 3)
	b.checkPiecesCnt(t, 6)
	b.checkContent("#3", t, "testing cach insertation")

	b.cacheInsertString(12, "ed")
	b.checkPiecesCnt(t, 6)
	b.checkContent("#4", t, "testing cached insertation")
}

func TestSimulateBackspace(t *testing.T) {
	b := NewBuffer([]byte("apples and oranges"))
	for i := 5; i > 0; i-- {
		b.cacheDelete(i, 1)
	}
	b.checkContent("#0", t, "a and oranges")
	b.Undo()
	b.checkContent("#1", t, "apples and oranges")
}

func TestSimulateDeleteKey(t *testing.T) {
	b := NewBuffer([]byte("apples and oranges"))
	for i := 0; i < 4; i++ {
		b.cacheDelete(7, 1)
	}
	b.checkContent("#0", t, "apples oranges")
	b.Undo()
	b.checkContent("#1", t, "apples and oranges")
}

func TestDelete(t *testing.T) {
	b := NewBuffer([]byte("and what is a dream?"))
	b.insertString(9, "exactly ")
	b.checkContent("#0", t, "and what exactly is a dream?")

	b.delete(22, 2000)
	b.checkContent("#1", t, "and what exactly is a ")
	b.insertString(22, "joke?")
	b.checkContent("#2", t, "and what exactly is a joke?")

	cases := []struct {
		pos, len int
		expected string
	}{
		{9, 8, "and what is a joke?"},
		{9, 13, "and what joke?"},
		{5, 6, "and wactly is a joke?"},
		{9, 14, "and what oke?"},
		{11, 3, "and what exly is a joke?"},
	}
	for _, c := range cases {
		b.delete(c.pos, c.len)
		b.checkContent("#3", t, c.expected)
		b.Undo()
		b.checkContent("#4", t, "and what exactly is a joke?")
	}
}

func TestGroupChanges(t *testing.T) {
	b := NewBuffer([]byte("group 1, group 2, group 3"))
	b.checkPiecesCnt(t, 3)
	// b.GroupChanges()

	b.cacheDelete(0, 6)
	b.checkContent("#0", t, "1, group 2, group 3")

	b.cacheDelete(3, 6)
	b.checkContent("#1", t, "1, 2, group 3")

	b.cacheDelete(6, 6)
	b.checkContent("#2", t, "1, 2, 3")

	b.Undo()
	b.checkContent("#3", t, "group 1, group 2, group 3")
	b.Undo()
	b.checkContent("#4", t, "group 1, group 2, group 3")

	b.Redo()
	b.checkContent("#5", t, "1, 2, 3")
	b.Redo()
	b.checkContent("#6", t, "1, 2, 3")
}

func TestSaving(t *testing.T) {
	b := NewBuffer(nil)

	b.checkModified(t, 1, false)
	b.insertString(0, "stars can frighten")
	b.checkModified(t, 2, true)

	b.Save()
	b.checkModified(t, 3, false)

	b.Undo()
	b.checkModified(t, 4, true)
	b.Redo()
	b.checkModified(t, 5, false)

	b.insertString(0, "Neptun, Titan, ")
	b.checkModified(t, 6, true)
	b.Undo()
	b.checkModified(t, 7, false)

	b.Redo()
	b.checkModified(t, 8, true)

	b.Save()
	b.checkModified(t, 9, false)

	b = NewBuffer([]byte("my book is closed"))
	b.checkModified(t, 10, false)

	b.insertString(17, ", I read no more")
	b.checkModified(t, 11, true)
	b.Undo()
	b.checkModified(t, 12, false)

	b.Redo()
	b.Save()
	b.checkModified(t, 13, false)

	b.Undo()
	b.Save()
	b.checkModified(t, 14, false)
}

func TestReader(t *testing.T) {
	b := NewBuffer(nil)
	b.insertString(0, "So many")
	b.insertString(7, " books,")
	b.insertString(14, " so little")
	b.insertString(24, " time.")
	b.checkContent("#0", t, "So many books, so little time.")

	cases := []struct {
		off, len int
		expected string
		err      error
	}{
		{0, 7, "So many", nil},
		{1, 11, "o many book", nil},
		{8, 4, "book", nil},
		{15, 20, "so little time.", io.EOF},
	}

	for _, c := range cases {
		data := make([]byte, c.len)
		n, err := b.ReadAt(data, int64(c.off))
		if err != c.err {
			t.Errorf("expected error %v, got %v", c.err, err)
		}
		if n != len(c.expected) {
			t.Errorf("n should be %d, got %d", len(c.expected), n)
		}
		if !bytes.Equal(data[:n], []byte(c.expected)) {
			t.Errorf("got '%s', want '%s'", data[:n], c.expected)
		}
	}
}

func (b *Buffer) checkPiecesCnt(t *testing.T, expected int) {
	if b.piecesCnt != expected {
		t.Errorf("got %d pieces, want %d", b.piecesCnt, expected)
	}
}

func (b *Buffer) checkContent(name string, t *testing.T, expected string) {
	c := b.allContent()
	if c != expected {
		t.Errorf("%s: got '%s', want '%s'", name, c, expected)
	}
}

func (t *Buffer) insertString(pos int, data string) {
	t.CommitChanges()
	t.cacheInsertString(pos, data)
}

func (t *Buffer) cacheInsertString(pos int, data string) {
	err := t.Insert(int64(pos), []byte(data))
	if err != nil {
		panic(err)
	}
}

func (t *Buffer) delete(pos, length int) {
	t.CommitChanges()
	t.cacheDelete(pos, length)
}

func (t *Buffer) cacheDelete(pos, length int) {
	t.Delete(int64(pos), int64(length))
}

func (t *Buffer) printPieces() {
	for p := t.begin; p != nil; p = p.next {
		prev, next := 0, 0
		if p.prev != nil {
			prev = p.prev.id
		}
		if p.next != nil {
			next = p.next.id
		}
		fmt.Printf("%d, p:%d, n:%d = %s\n", p.id, prev, next, string(p.data))
	}
	fmt.Println()
}

func (b *Buffer) checkModified(t *testing.T, id int, expected bool) {
	if b.Modified() != expected {
		if expected {
			t.Errorf("#%d should be modified", id)
		} else {
			t.Errorf("#%d should not be modified", id)
		}
	}
}

func (t *Buffer) allContent() string {
	var data []byte
	p := t.begin.next
	for p != t.end {
		data = append(data, p.data...)
		p = p.next

	}
	return string(data)
}
