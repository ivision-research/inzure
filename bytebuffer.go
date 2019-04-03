package inzure

import (
	"errors"
	"fmt"
	"io"
)

type byteBuffer struct {
	buf  []byte
	idx  int
	size int
}

func newByteBuffer() *byteBuffer {
	return &byteBuffer{
		buf:  make([]byte, 0),
		idx:  0,
		size: 0,
	}
}

func newByteBufferFromBytes(b []byte) *byteBuffer {
	return &byteBuffer{
		buf:  b,
		idx:  0,
		size: len(b),
	}
}

func (b *byteBuffer) Len() int {
	return b.size
}

func (b *byteBuffer) Reset() {
	b.size = 0
	b.idx = 0
	b.buf = b.buf[:0]
}

func (b *byteBuffer) Read(into []byte) (int, error) {
	if into == nil {
		return 0, errors.New("tried to read into null buffer")
	}
	if len(into) == 0 {
		return 0, errors.New("tried to read into 0 length slice")
	}
	if b.idx >= b.size {
		return 0, io.EOF
	}
	l := len(into)
	rem := b.size - b.idx
	// can't fill it, next call will be 0, io.EOF
	if rem < l {
		l = rem
	}
	copied := copy(into, b.buf[b.idx:b.idx+l])
	b.idx += copied
	return copied, nil
}

func (b *byteBuffer) Write(from []byte) (int, error) {
	if from == nil {
		return 0, errors.New("tried to write nil slice")
	}
	l := len(from)
	b.size += l
	b.buf = append(b.buf, from...)
	return l, nil
}

func (b *byteBuffer) Seek(offset int64, whence int) (int64, error) {
	ioff := int(offset)
	switch whence {
	case io.SeekStart:
		if ioff >= b.size {
			return 0, fmt.Errorf(
				"tried to seek past EOF (%d > %d)", ioff, b.size,
			)
		} else if ioff < 0 {
			return 0, errors.New("tried to seek past beginning")
		} else {
			b.idx = ioff
			return int64(b.idx), nil
		}
	case io.SeekCurrent:
		change := b.idx + ioff
		if change < 0 {
			return 0, errors.New("tried to seek past beginning")
		} else if change >= b.size {
			return 0, errors.New("tried to seek past EOF")
		} else {
			b.idx = change
			return int64(b.idx), nil
		}
	case io.SeekEnd:
		if ioff > 0 {
			return 0, errors.New("tried to seek past EOF")
		} else if ioff > b.size {
			return 0, fmt.Errorf(
				"tried to seek past beginning (%d > %d)", ioff, b.size,
			)
		} else {
			b.idx = b.size - ioff
			return int64(b.idx), nil
		}
	}
	return int64(b.idx), fmt.Errorf("unsupported whence %d", whence)
}

func (b *byteBuffer) Close() error {
	// noop
	return nil
}
