package session

import "io"

// Session abstracts a terminal session's I/O and resize control.
// A local session wraps a PTY; a remote session wraps two VFS file handles.
type Session interface {
	io.ReadWriteCloser
	Resize(rows, cols int) error
}
