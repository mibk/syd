package textutil

import (
	"strings"
	"testing"
)

func TestFindingLineStartAndEnd(t *testing.T) {
	r := strings.NewReader(`I
have
nothing
to say.
I just want to make sure that this line is greater than the bufSize which is 50.`)

	tests := []struct {
		off   int64
		start int64
		end   int64
	}{
		{0, 0, 2},
		{1, 0, 2},
		{2, 2, 7},
		{5, 2, 7},
		{10, 7, 15},
		{23, 23, 103},
		{83, 23, 103},
	}

	for _, test := range tests {
		if got := FindLineStart(r, test.off); test.start != got {
			t.Errorf("find line start: got %v, want %v", got, test.start)
		}
		if got := FindLineEnd(r, test.off); test.end != got {
			t.Errorf("find line end: got %v, want %v", got, test.end)
		}
	}
}

func TestFindIndentOffset(t *testing.T) {
	r := strings.NewReader("What's\n\tthe    ugliest  \t part of your body? \t\t")

	tests := []struct {
		off    int64
		indent int64
	}{
		{0, 0},
		{4, 4},
		{6, 6},
		{7, 8},
		{11, 15},
		{22, 26},
		{24, 26},
		{28, 28},
		{44, 47},
	}

	for _, test := range tests {
		if ind := FindIndentOffset(r, test.off); ind != test.indent {
			t.Errorf("got %d, want %d", ind, test.indent)
		}
	}
}
