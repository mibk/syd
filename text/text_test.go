package text

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

func TestOverall(t *testing.T) {
	txt := New(nil)
	txt.checkPiecesCnt(t, 2)
	txt.checkContent("#0", t, "")

	txt.insertString(0, "")
	txt.checkPiecesCnt(t, 2)
	txt.checkContent("#1", t, "")

	txt.insertString(0, "All work makes John a dull boy")
	txt.checkPiecesCnt(t, 3)
	txt.checkContent("#2", t, "All work makes John a dull boy")

	txt.insertString(9, "and no playing ")
	txt.checkPiecesCnt(t, 6)
	txt.checkContent("#3", t, "All work and no playing makes John a dull boy")

	txt.CommitChanges()
	// Also check that multiple change commits don't create empty changes.
	txt.CommitChanges()
	txt.Delete(20, 14)
	txt.checkContent("#4", t, "All work and no play a dull boy")

	txt.insertString(20, " makes Jack")
	txt.checkContent("#5", t, "All work and no play makes Jack a dull boy")

	txt.Undo()
	txt.checkContent("#6", t, "All work and no play a dull boy")
	txt.Undo()
	txt.checkContent("#7", t, "All work and no playing makes John a dull boy")
	txt.Undo()
	txt.checkContent("#8", t, "All work makes John a dull boy")

	txt.Redo()
	txt.checkContent("#9", t, "All work and no playing makes John a dull boy")
	txt.Redo()
	txt.checkContent("#10", t, "All work and no play a dull boy")
	txt.Redo()
	txt.checkContent("#11", t, "All work and no play makes Jack a dull boy")
	txt.Redo()
	txt.checkContent("#12", t, "All work and no play makes Jack a dull boy")
}

func TestCacheInsertAndDelete(t *testing.T) {
	txt := New([]byte("testing insertation"))
	txt.checkPiecesCnt(t, 3)
	txt.checkContent("#0", t, "testing insertation")

	txt.cacheInsertString(8, "caching")
	txt.checkPiecesCnt(t, 6)
	txt.checkContent("#1", t, "testing cachinginsertation")

	txt.cacheInsertString(15, " ")
	txt.checkPiecesCnt(t, 6)
	txt.checkContent("#2", t, "testing caching insertation")

	txt.cacheDelete(12, 3)
	txt.checkPiecesCnt(t, 6)
	txt.checkContent("#3", t, "testing cach insertation")

	txt.cacheInsertString(12, "ed")
	txt.checkPiecesCnt(t, 6)
	txt.checkContent("#4", t, "testing cached insertation")
}

func TestSimulateBackspace(t *testing.T) {
	txt := New([]byte("apples and oranges"))
	for i := 5; i > 0; i-- {
		txt.cacheDelete(i, 1)
	}
	txt.checkContent("#0", t, "a and oranges")
	txt.Undo()
	txt.checkContent("#1", t, "apples and oranges")
}

func TestSimulateDeleteKey(t *testing.T) {
	txt := New([]byte("apples and oranges"))
	for i := 0; i < 4; i++ {
		txt.cacheDelete(7, 1)
	}
	txt.checkContent("#0", t, "apples oranges")
	txt.Undo()
	txt.checkContent("#1", t, "apples and oranges")
}

func TestDelete(t *testing.T) {
	txt := New([]byte("and what is a dream?"))
	txt.insertString(9, "exactly ")
	txt.checkContent("#0", t, "and what exactly is a dream?")

	txt.delete(22, 2000)
	txt.checkContent("#1", t, "and what exactly is a ")
	txt.insertString(22, "joke?")
	txt.checkContent("#2", t, "and what exactly is a joke?")

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
		txt.delete(c.pos, c.len)
		txt.checkContent("#3", t, c.expected)
		txt.Undo()
		txt.checkContent("#4", t, "and what exactly is a joke?")
	}
}

func TestGroupChanges(t *testing.T) {
	txt := New([]byte("group 1, group 2, group 3"))
	txt.checkPiecesCnt(t, 3)
	// txt.GroupChanges()

	txt.cacheDelete(0, 6)
	txt.checkContent("#0", t, "1, group 2, group 3")

	txt.cacheDelete(3, 6)
	txt.checkContent("#1", t, "1, 2, group 3")

	txt.cacheDelete(6, 6)
	txt.checkContent("#2", t, "1, 2, 3")

	txt.Undo()
	txt.checkContent("#3", t, "group 1, group 2, group 3")
	txt.Undo()
	txt.checkContent("#4", t, "group 1, group 2, group 3")

	txt.Redo()
	txt.checkContent("#5", t, "1, 2, 3")
	txt.Redo()
	txt.checkContent("#6", t, "1, 2, 3")
}

func TestSaving(t *testing.T) {
	txt := New(nil)

	txt.checkModified(t, 1, false)
	txt.insertString(0, "stars can frighten")
	txt.checkModified(t, 2, true)

	txt.Save()
	txt.checkModified(t, 3, false)

	txt.Undo()
	txt.checkModified(t, 4, true)
	txt.Redo()
	txt.checkModified(t, 5, false)

	txt.insertString(0, "Neptun, Titan, ")
	txt.checkModified(t, 6, true)
	txt.Undo()
	txt.checkModified(t, 7, false)

	txt.Redo()
	txt.checkModified(t, 8, true)

	txt.Save()
	txt.checkModified(t, 9, false)

	txt = New([]byte("my book is closed"))
	txt.checkModified(t, 10, false)

	txt.insertString(17, ", I read no more")
	txt.checkModified(t, 11, true)
	txt.Undo()
	txt.checkModified(t, 12, false)

	txt.Redo()
	txt.Save()
	txt.checkModified(t, 13, false)

	txt.Undo()
	txt.Save()
	txt.checkModified(t, 14, false)
}

func TestReader(t *testing.T) {
	txt := New(nil)
	txt.insertString(0, "So many")
	txt.insertString(7, " books,")
	txt.insertString(14, " so little")
	txt.insertString(24, " time.")
	txt.checkContent("#0", t, "So many books, so little time.")

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
		n, err := txt.ReadAt(data, int64(c.off))
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

func (txt *Text) checkPiecesCnt(t *testing.T, expected int) {
	if txt.piecesCnt != expected {
		t.Errorf("got %d pieces, want %d", txt.piecesCnt, expected)
	}
}

func (txt *Text) checkContent(name string, t *testing.T, expected string) {
	c := txt.allContent()
	if c != expected {
		t.Errorf("%s: got '%s', want '%s'", name, c, expected)
	}
}

func (t *Text) insertString(pos int, data string) {
	t.CommitChanges()
	t.cacheInsertString(pos, data)
}

func (t *Text) cacheInsertString(pos int, data string) {
	err := t.Insert(pos, []byte(data))
	if err != nil {
		panic(err)
	}
}

func (t *Text) delete(pos, length int) {
	t.CommitChanges()
	t.cacheDelete(pos, length)
}

func (t *Text) cacheDelete(pos, length int) {
	t.Delete(pos, length)
}

func (t *Text) printPieces() {
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

func (txt *Text) checkModified(t *testing.T, id int, expected bool) {
	if txt.Modified() != expected {
		if expected {
			t.Errorf("#%d should be modified", id)
		} else {
			t.Errorf("#%d should not be modified", id)
		}
	}
}

func (t *Text) allContent() string {
	var data []byte
	p := t.begin.next
	for p != t.end {
		data = append(data, p.data...)
		p = p.next

	}
	return string(data)
}
