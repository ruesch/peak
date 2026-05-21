package main

import (
	"io"
	"sync"
)

// ExternalPTY is a session.Session backed by external I/O rather than a local
// process. TermView calls Write (keypresses → accumulate for external reader)
// and Read (blocks until external program writes terminal output). The window's
// /io file is the external side: WriteAt feeds the terminal emulator; ReadAt
// streams user input with blocking offset semantics.
type ExternalPTY struct {
	// inBuf grows as the user types; external program reads via ReadInput.
	inMu   sync.Mutex
	inCond *sync.Cond
	inBuf  []byte
	inDone bool

	// outBuf is consumed by TermView's Read; external program writes via WriteOutput.
	outMu   sync.Mutex
	outCond *sync.Cond
	outBuf  []byte
	outDone bool

	// onResize is called on the main goroutine when the window is resized.
	onResize func(rows, cols int)
}

func newExternalPTY() *ExternalPTY {
	p := &ExternalPTY{}
	p.inCond = sync.NewCond(&p.inMu)
	p.outCond = sync.NewCond(&p.outMu)
	return p
}

// --- session.Session (TermView side) ---

// Read blocks until terminal output is available, then consumes it.
func (p *ExternalPTY) Read(buf []byte) (int, error) {
	p.outMu.Lock()
	defer p.outMu.Unlock()
	for len(p.outBuf) == 0 && !p.outDone {
		p.outCond.Wait()
	}
	if len(p.outBuf) == 0 {
		return 0, io.EOF
	}
	n := copy(buf, p.outBuf)
	p.outBuf = p.outBuf[n:]
	return n, nil
}

// Write accumulates user keypresses for the external reader.
func (p *ExternalPTY) Write(data []byte) (int, error) {
	p.inMu.Lock()
	p.inBuf = append(p.inBuf, data...)
	p.inCond.Broadcast()
	p.inMu.Unlock()
	return len(data), nil
}

// Resize fires the onResize callback, allowing the external program to relay
// the new size to a remote PTY.
func (p *ExternalPTY) Resize(rows, cols int) error {
	if p.onResize != nil {
		p.onResize(rows, cols)
	}
	return nil
}

// Close unblocks all pending reads on both sides.
func (p *ExternalPTY) Close() error {
	p.inMu.Lock()
	p.inDone = true
	p.inCond.Broadcast()
	p.inMu.Unlock()

	p.outMu.Lock()
	p.outDone = true
	p.outCond.Broadcast()
	p.outMu.Unlock()
	return nil
}

// --- External side (winIoFile) ---

// ReadInput blocks until user input is available at the given offset, then
// returns bytes starting there. Used by the window's /io file ReadAt.
func (p *ExternalPTY) ReadInput(buf []byte, off int64) (int, error) {
	p.inMu.Lock()
	defer p.inMu.Unlock()
	for int64(len(p.inBuf)) <= off && !p.inDone {
		p.inCond.Wait()
	}
	if int64(len(p.inBuf)) <= off {
		return 0, io.EOF
	}
	n := copy(buf, p.inBuf[off:])
	return n, nil
}

// WriteOutput feeds terminal output bytes into the emulator. Used by the
// window's /io file WriteAt.
func (p *ExternalPTY) WriteOutput(data []byte) (int, error) {
	p.outMu.Lock()
	p.outBuf = append(p.outBuf, data...)
	p.outCond.Broadcast()
	p.outMu.Unlock()
	return len(data), nil
}
