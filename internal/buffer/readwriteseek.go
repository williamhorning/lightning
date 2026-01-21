// Package buffer implements a bytes.Buffer-like interface but also seeks.
package buffer

import "io"

// Buffer is a read write seeker.
type Buffer struct {
	data []byte
	pos  int
}

// Read reads bytes from the buffer.
func (b *Buffer) Read(p []byte) (int, error) {
	if b.pos >= len(b.data) {
		return 0, io.EOF
	}

	n := copy(p, b.data[b.pos:])
	b.pos += n

	return n, nil
}

// Write appends or overwrites bytes in the buffer, growing it if necessary.
func (b *Buffer) Write(bytes []byte) (int, error) {
	endPos := b.pos + len(bytes)

	if endPos > len(b.data) {
		diff := endPos - len(b.data)
		b.data = append(b.data, make([]byte, diff)...)
	}

	n := copy(b.data[b.pos:], bytes)
	b.pos += n

	return n, nil
}

// Seek changes the position of the buffer.
func (b *Buffer) Seek(offset int64, whence int) (int64, error) {
	var newPos int

	switch whence {
	case io.SeekStart:
		newPos = int(offset)
	case io.SeekCurrent:
		newPos = b.pos + int(offset)
	case io.SeekEnd:
		newPos = len(b.data) + int(offset)
	default:
		return 0, seekError{}
	}

	if newPos < 0 {
		return 0, seekError{}
	}

	b.pos = newPos

	return int64(newPos), nil
}

type seekError struct{}

func (seekError) Error() string {
	return "invalid seek. don't do that."
}
