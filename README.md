# undo

Package undo provides methods for undoable/redoable text manipulation.
Modifications are preformed with either insert or delete.

This package and documentation are based on the Undo structure in the syd editor. For further information please visit
[https://github.com/mibk/syd](https://github.com/mibk/syd)

## Insertion

When inserting new data there are 2 cases to consider:

1. the insertion point falls into the middle of an existing piece which
is replaced by three new pieces:

```go
/-+ --> +---------------+ --> +-\
| |     | existing text |     | |
\-+ <-- +---------------+ <-- +-/
                   ^
                   insertion point for "demo "

/-+ --> +---------+ --> +-----+ --> +-----+ --> +-\
| |     | existing|     |demo |     |text |     | |
\-+ <-- +---------+ <-- +-----+ <-- +-----+ <-- +-/
```

2. it falls at a piece boundary:

```go
/-+ --> +---------------+ --> +-\
| |     | existing text |     | |
\-+ <-- +---------------+ <-- +-/
      ^
      insertion point for "short"

/-+ --> +-----+ --> +---------------+ --> +-\
| |     |short|     | existing text |     | |
\-+ <-- +-----+ <-- +---------------+ <-- +-/
```

## Deletion

The delete operation can either start/stop midway through a piece or at
a boundary. In the former case a new piece is created to represent the
remaining text before/after the modification point.

```go
/-+ --> +---------+ --> +-----+ --> +-----+ --> +-\
| |     | existing|     |demo |     |text |     | |
\-+ <-- +---------+ <-- +-----+ <-- +-----+ <-- +-/
             ^                         ^
             |------ delete range -----|

/-+ --> +----+ --> +--+ --> +-\
| |     | exi|     |t |     | |
\-+ <-- +----+ <-- +--+ <-- +-/
```

## Changes

Undoing and redoing works with actions (action is a group of changes: insertions
and deletions). An action is represented by any operations between two calls of
Commit method. Anything that happens between these two calls is a part of that
particular action.

---
