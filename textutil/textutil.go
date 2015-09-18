package textutil

import (
	"bytes"
	"io"
)

const bufSize = 50

func FindLineStart(r io.ReaderAt, off int64) int64 {
	if off <= 0 {
		return 0
	}
	data := make([]byte, bufSize)
	for {
		off -= bufSize
		max := bufSize
		if off < 0 {
			max += int(off)
			off = 0
		}
		n, err := r.ReadAt(data[:max], off)
		if err != nil {
			panic(err)
		}
		i := bytes.LastIndexByte(data[:n], '\n')
		if i != -1 {
			return off + int64(i) + 1
		} else if off == 0 {
			return 0
		}
	}
}

func FindLineEnd(r io.ReaderAt, off int64) int64 {
	data := make([]byte, bufSize)
	for {
		n, err := r.ReadAt(data, off)
		if err != nil && err != io.EOF {
			panic(err)
		}
		i := bytes.IndexByte(data[:n], '\n')
		if i != -1 {
			return off + int64(i) + 1
		} else if err == io.EOF {
			return off + int64(n)
		}
		off += bufSize
	}
}
