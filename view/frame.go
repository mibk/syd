package view

const (
	ColQ0 = -1
	ColQ1 = -2
)

type Frame struct {
	Lines       [][]rune
	Line0, Col0 int
	Line1, Col1 int
	WantCol     int
	Nchars      int
}

func (f Frame) NextNewLine(n int) int {
	c := 0
	for _, l := range f.Lines {
		c += len(l) + 1 // + '\n'
		n--
		if n == 0 {
			goto NotLastLine
		}
	}
	c-- // last line doesn't contain '\n'
NotLastLine:
	return c
}

// CharsToXY returns the number of characters from beginning
// to the position given by x and y.
func (f Frame) CharsToXY(x, y int) int {
	if y >= len(f.Lines) {
		return f.Nchars + 1
	}
	var p int
	for n, l := range f.Lines {
		if n == y {
			return p + CharsToX(l, x)
		}
		p += len(l) + 1 // + '\n'
	}
	panic("shouldn't happen")
}

// CharsToX returns the number of characters from beginning
// to the position x.
func CharsToX(s []rune, x int) int {
	var w int
	for i, r := range s {
		if r == '\t' {
			w += TabWidthForCol(w)
		} else {
			w += 1
		}
		if w > x {
			return i
		}
	}
	return len(s)
}
