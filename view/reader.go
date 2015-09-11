package view

import "io"

// ReaderFrom accepts an io.ReaderAt and returns the io.Reader
// that can read from the offset to the EOF.
func ReaderFrom(r io.ReaderAt, offset int64) io.Reader {
	return &Reader{r, offset}
}

type Reader struct {
	readerAt io.ReaderAt
	off      int64
}

func (r *Reader) Read(data []byte) (n int, err error) {
	n, err = r.readerAt.ReadAt(data, r.off)
	r.off += int64(n)
	if n != 0 && err == io.EOF {
		err = nil
	}
	return
}
