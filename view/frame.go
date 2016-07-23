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
