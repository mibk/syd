package core

import (
	"io"
	"os"

	mmap "github.com/edsrzf/mmap-go"
)

type Content interface {
	Bytes() []byte
	io.Closer
}

type BytesContent []byte

func (b BytesContent) Bytes() []byte {
	return b
}

func (b BytesContent) Close() error {
	return nil
}

func Mmap(f *os.File) (Content, error) {
	m, err := mmap.Map(f, 0, 0)
	if err != nil {
		return nil, err
	}
	return &_mmap{m}, nil
}

type _mmap struct {
	m mmap.MMap
}

func (mm *_mmap) Bytes() []byte {
	return mm.m
}

func (mm *_mmap) Close() error {
	return mm.m.Unmap()
}
