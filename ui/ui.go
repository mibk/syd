package ui

type Window interface {
	// Init initialises the UI.
	Init() error

	// Close safely destroys the UI resources.
	Close() error

	// Size returns size of the window.
	Size() (w, h int)

	// Clear clears the frame buffer and enables the use of WriteRune.
	Clear()

	// Select sets which characters should be selected. If p0 == p1, cursor
	// is placed instead. Select must be called before calls to WriteRune.
	Select(p0, p1 int)

	// WriteRune writes rune to the frame buffer. When there is no more
	// space for characters to be displayed, io.EOF is return.
	WriteRune(r rune) error

	// Flushed flushes the frame buffer, making the changes to the frame
	// buffer visible.
	Flush()

	// Frame returns the underlying frame buffer.
	Frame() Frame
}

const (
	ColQ0 = -1
	ColQ1 = -2
)

type Frame interface {
	// Nchars returns the number of characters in the frame.
	Nchars() int

	// SelectionLines return the line numbers of the begginging and end
	// of the selection.
	SelectionLines() (int, int)

	// CharsUntilXY returns the number of characters from beginning
	// to the position given by x, y.
	CharsUntilXY(x, y int) int

	// MaxLines returns the maximal number of lines in the frame.
	MaxLines() int

	// Lines return the number of actual lines in the frame.
	Lines() int

	WantCol() int
	SetWantCol(int)
}
