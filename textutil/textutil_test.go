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
