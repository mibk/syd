// The majority of parts of this file is based on the text manipulation in
// the vis editor by Marc André Tanner and are available under the copyright
// bellow.  For further information please visit http://repo.or.cz/w/vis.git
// or https://github.com/martanne/vis.

// Copyright (c) 2014 Marc André Tanner <mat at brain-dump.org>
//
// Permission to use, copy, modify, and/or distribute this software for any
// purpose with or without fee is hereby granted, provided that the above
// copyright notice and this permission notice appear in all copies.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
// WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
// ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
// WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
// ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
// OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

// Package text provides methods for undoable/redoable text manipulation.
// Modifications are made by two operations: insert or delete.
//
// The package is based on the text manipulation in the vis editor. (Some parts are
// pure ports.) For further information please visit
// 	https://github.com/martanne/vis.
//
//
// Insertation
//
// When inserting new data there are 2 cases to consider:
//
// 1. the insertion point falls into the middle of an exisiting piece which
// is replaced by three new pieces:
//
//	/-+ --> +---------------+ --> +-\
//	| |     | existing text |     | |
//	\-+ <-- +---------------+ <-- +-/
//	                   ^
//	                   insertion point for "demo "
//
//	/-+ --> +---------+ --> +-----+ --> +-----+ --> +-\
//	| |     | existing|     |demo |     |text |     | |
//	\-+ <-- +---------+ <-- +-----+ <-- +-----+ <-- +-/
//
// 2. it falls at a piece boundary:
//
//	/-+ --> +---------------+ --> +-\
//	| |     | existing text |     | |
//	\-+ <-- +---------------+ <-- +-/
//	      ^
//	      insertion point for "short"
//
//	/-+ --> +-----+ --> +---------------+ --> +-\
//	| |     |short|     | existing text |     | |
//	\-+ <-- +-----+ <-- +---------------+ <-- +-/
//
//
// Deletion
//
// The delete operation can either start/stop midway through a piece or at
// a boundary. In the former case a new piece is created to represent the
// remaining text before/after the modification point.
//
//	/-+ --> +---------+ --> +-----+ --> +-----+ --> +-\
//	| |     | existing|     |demo |     |text |     | |
//	\-+ <-- +---------+ <-- +-----+ <-- +-----+ <-- +-/
//	             ^                         ^
//	             |------ delete range -----|
//
//	/-+ --> +----+ --> +--+ --> +-\
//	| |     | exi|     |t |     | |
//	\-+ <-- +----+ <-- +--+ <-- +-/
//
package text

import (
	"errors"
	"io"
	"time"
)

var ErrWrongPos = errors.New("position is greater than text size")

// piece represents a piece of the text. All active pieces chained together form
// the whole content of the text.
type piece struct {
	id         int
	prev, next *piece
	data       []byte
}

func (p *piece) len() int {
	return len(p.data)
}

func (p *piece) insert(pos int, data []byte) {
	p.data = append(p.data[:pos], append(data, p.data[pos:]...)...)
}

func (p *piece) delete(pos, length int) bool {
	if pos+length > len(p.data) {
		return false
	}
	p.data = append(p.data[:pos], p.data[pos+length:]...)
	return true
}

// span holds a certain range of pieces. Changes to the document are allways
// performed by swapping out an existing span with a new one.
type span struct {
	start, end *piece // start/end of the span
	len        int    // the sum of the lengths of the pieces which form this span
}

// change keeps all needed information to redo/undo an insertion/deletion.
type change struct {
	old span // all pieces which are being modified/swapped out by the change
	new span // all pieces which are introduced/swapped int by the change
	pos int  // absolute position at which the change occured
}

// action is a list of changes which are used to undo/redo all modifications.
type action struct {
	changes []*change
	time    time.Time // when the first change of this action was performed
}

func newSpan(start, end *piece) span {
	s := span{start: start, end: end}
	for p := start; p != nil; p = p.next {
		s.len += p.len()
		if p == end {
			break
		}
	}
	return s
}

// swapSpans swaps out an old span and replace it with a new one.
//  - If old is an empty span do not remove anything, just insert the new one.
//  - If new is an empty span do not insert anything, just remove the old one.
func swapSpans(old, new span) {
	if old.len == 0 && new.len == 0 {
		return
	} else if old.len == 0 {
		// insert new span
		new.start.prev.next = new.start
		new.end.next.prev = new.end
	} else if new.len == 0 {
		// delete old span
		old.start.prev.next = old.end.next
		old.end.next.prev = old.start.prev
	} else {
		// replace old with new
		old.start.prev.next = new.start
		old.end.next.prev = new.end
	}
}

// Text is the representation of a structure capable of two operations: inserting or
// deleting. All operations could be unlimitedly undone or redone.
type Text struct {
	piecesCnt   int    // number of pieces allocated
	begin, end  *piece // sentinel nodes which always exists but don't hold any data
	cachedPiece *piece // most recently modified piece

	actions       []*action // stack holding all actions performed to the file
	head          int       // index for the next action to add
	groupChanges  bool
	currentAction *action // action for the current change group
	savedAction   *action
}

// New initializes a new text with the given content as a starting point.
// To start with an empty text pass nil as a content.
func New(content []byte) *Text {
	// give the actions stack some default capacity
	t := &Text{actions: make([]*action, 0, 100)}

	t.begin = t.newEmptyPiece()
	t.end = t.newPiece(nil, t.begin, nil)
	t.begin.next = t.end

	if content != nil {
		p := t.newPiece(content, t.begin, t.end)
		t.begin.next = p
		t.end.prev = p
	}
	return t
}

func (t *Text) newPiece(data []byte, prev, next *piece) *piece {
	t.piecesCnt++
	return &piece{
		id:   t.piecesCnt,
		prev: prev,
		next: next,
		data: data,
	}
}

func (t *Text) newEmptyPiece() *piece {
	return t.newPiece(nil, nil, nil)
}

// findPiece returns the piece holding the text at the byte offset pos. If pos happens
// to be at a piece boundary i.e. the first byte of a piece then the previous piece
// to the left is returned with an offset of piece's length.
//
// If pos is zero, the begin sentinel piece is returned.
func (t *Text) findPiece(pos int) (p *piece, offset int) {
	cur := 0
	for p = t.begin; p.next != nil; p = p.next {
		if cur <= pos && pos <= cur+p.len() {
			return p, pos - cur
		}
		cur += p.len()
	}
	return nil, 0
}

// newChange is associated with the current action or a newly allocated one if
// none exists.
func (t *Text) newChange(pos int) *change {
	a := t.currentAction
	if a == nil {
		a = t.newAction()
		if t.groupChanges {
			t.cachedPiece = nil
			t.currentAction = a
		}
	}
	c := &change{pos: pos}
	a.changes = append(a.changes, c)
	return c
}

// newAction creates a new action and throws away all undone actions.
func (t *Text) newAction() *action {
	a := &action{time: time.Now()}
	t.actions = append(t.actions[:t.head], a)
	t.head++
	return a
}

// Insert inserts the data to the pos in the text. An error is return when the
// given pos is invalid.
func (t *Text) Insert(pos int, data []byte) error {
	if len(data) == 0 {
		return nil
	}

	p, offset := t.findPiece(pos)
	if p == nil {
		return ErrWrongPos
	} else if p == t.cachedPiece {
		// just update the last inserted piece
		p.insert(offset, data)
		return nil
	}

	c := t.newChange(pos)
	var pnew *piece
	if offset == p.len() {
		// Insert between two existing pieces, hence there is nothing to
		// remove, just add a new piece holding the extra text.
		pnew = t.newPiece(data, p, p.next)
		c.new = newSpan(pnew, pnew)
		c.old = newSpan(nil, nil)
	} else {
		// Insert into middle of an existing piece, therefore split the old
		// piece. That is we have 3 new pieces one containing the content
		// before the insertion point then one holding the newly inserted
		// text and one holding the content after the insertion point.
		before := t.newPiece(p.data[:offset], p.prev, nil)
		pnew = t.newPiece(data, before, nil)
		after := t.newPiece(p.data[offset:], pnew, p.next)
		before.next = pnew
		pnew.next = after
		c.new = newSpan(before, after)
		c.old = newSpan(p, p)
	}

	t.cachedPiece = pnew
	swapSpans(c.old, c.new)
	return nil
}

// Delete deletes the portion of the text at the pos of given length. An error
// is returned if the portion isn't in the range of the text. If the length exceeds
// the size of the text, the portions from the pos to the end of the text will
// be deleted.
func (t *Text) Delete(pos, length int) error {
	if length <= 0 {
		return nil
	}

	p, offset := t.findPiece(pos)
	if p == nil {
		return ErrWrongPos
	} else if p == t.cachedPiece {
		// try to update the last inserted piece if the length doesn't exceeds
		if p.delete(offset, length) {
			return nil
		}
	}
	t.cachedPiece = nil

	var cur int // how much has already been deleted
	midwayStart, midwayEnd := false, false

	var before, after *piece // unmodified pieces before/after deletion point
	var start, end *piece    // span which is removed

	if offset == p.len() {
		// deletion starts at a piece boundary
		before = p
		start = p.next
	} else {
		// deletion starts midway through a piece
		midwayStart = true
		cur = p.len() - offset
		start = p
		before = t.newEmptyPiece()
	}

	// skip all pieces which fall into deletion range
	for cur < length {
		if p.next == t.end {
			// delete all
			length = cur
			break
		}
		p = p.next
		if p == nil {
		}
		cur += p.len()
	}

	if cur == length {
		// deletion stops at a piece boundary
		end = p
		after = p.next
	} else {
		// deletion stops midway through a piece
		midwayEnd = true
		end = p
		after = t.newPiece(p.data[p.len()-cur+length:], before, p.next)
	}

	var newStart, newEnd *piece
	if midwayStart {
		// we finally know which piece follows our newly allocated before piece
		before.data = start.data[:offset]
		before.prev, before.next = start.prev, after

		newStart = before
		if !midwayEnd {
			newEnd = before
		}
	}
	if midwayEnd {
		newEnd = after
		if !midwayStart {
			newStart = after
		}
	}

	c := t.newChange(pos)
	c.new = newSpan(newStart, newEnd)
	c.old = newSpan(start, end)
	swapSpans(c.old, c.new)

	return nil
}

// Undo reverts the last performed action. It return the position in bytes
// which the action occured on. If there is no action to undo, returned
// position would be -1.
func (t *Text) Undo() int {
	t.CommitChanges()
	a := t.unshiftAction()
	if a == nil {
		return -1
	}
	var pos int
	for i := len(a.changes) - 1; i >= 0; i-- {
		c := a.changes[i]
		swapSpans(c.new, c.old)
		pos = c.pos
	}

	return pos
}

func (t *Text) unshiftAction() *action {
	if t.head == 0 {
		return nil
	}
	t.head--
	return t.actions[t.head]
}

// Redo repeats the last undone action. It return the position in bytes
// which the action occured on. If there is no action to redo, returned
// position would be -1.
func (t *Text) Redo() int {
	t.CommitChanges()
	a := t.shiftAction()
	if a == nil {
		return -1
	}
	var pos int
	for _, c := range a.changes {
		swapSpans(c.old, c.new)
		pos = c.pos
	}
	return pos
}

func (t *Text) shiftAction() *action {
	if t.head > len(t.actions)-1 {
		return nil
	}
	t.head++
	return t.actions[t.head-1]
}

// GroupChanges causes that all following changes would be a part of only
// one action, until a CommitChanges is called. All changes of this action
// are undone/redone together.
func (t *Text) GroupChanges() {
	t.groupChanges = true
}

// CommitChanges commits the current action. All following changes won't be
// a part of this action.
func (t *Text) CommitChanges() {
	t.groupChanges = false
	t.currentAction = nil
	t.cachedPiece = nil
}

// Save the current state.
func (t *Text) Save() {
	if t.head > 0 {
		t.savedAction = t.actions[t.head-1]
	}
}

// Modified reports whether the current state of t is different from the one
// in the time of calling Save.
func (t *Text) Modified() bool {
	return t.head > 0 && t.savedAction == t.actions[t.head-1]
}

func (t *Text) GetReader() *Reader {
	return &Reader{t}
}

type Reader struct {
	text *Text
}

func (r *Reader) ReadAt(data []byte, off int64) (n int, err error) {
	p := r.text.begin
	for ; p != nil; p = p.next {
		if off < int64(p.len()) {
			break
		}
		off -= int64(p.len())
	}
	if p == nil {
		if off == 0 {
			return 0, io.EOF
		}
		return 0, ErrWrongPos
	}

	for n < len(data) && p != nil {
		n += copy(data[n:], p.data[off:])
		p = p.next
		off = 0
	}
	if n < len(data) {
		return n, io.EOF
	}
	return n, nil
}
