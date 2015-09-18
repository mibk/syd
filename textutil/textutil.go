package textutil

import (
	"bytes"
	"io"
)

const bufSize = 50

// FindLineStart returns the offset of the first occurence of the \n rune
// to the left, or 0, in the r.
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

// FindLineEnd returns the offset of the first occurence of the rune behind \n,
// or EOF, in the r.
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

// FindIndent returns the offset of the first non-blank characeter from
// the off in the r. If there is no non-blank character, the size
// of the r is returned. Only ' ' and '\t' are considered as blank
// characters.
func FindIndentOffset(r io.ReaderAt, off int64) int64 {
	data := make([]byte, bufSize)
	ioffset := off
	for {
		n, err := r.ReadAt(data, off)
		if err != nil && err != io.EOF {
			panic(err)
		}
		for i := 0; i < n; i++ {
			if data[i] != ' ' && data[i] != '\t' {
				return ioffset + int64(i)
			}
		}
		ioffset += int64(n)
		if err == io.EOF {
			return ioffset
		}
	}
}
