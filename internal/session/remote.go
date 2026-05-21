package session

import (
	"fmt"

	"github.com/aleksana/peak/internal/vfs/afero"
)

// RemoteSession implements Session over a pair of VFS files obtained from a
// mounted filesystem's /new protocol: io carries the PTY byte stream,
// ctl accepts "resize WxH" and "kill" commands.
type RemoteSession struct {
	Io  afero.File
	Ctl afero.File
}

func NewRemote(io, ctl afero.File) *RemoteSession {
	return &RemoteSession{Io: io, Ctl: ctl}
}

func (s *RemoteSession) Read(p []byte) (int, error)  { return s.Io.Read(p) }
func (s *RemoteSession) Write(p []byte) (int, error) { return s.Io.Write(p) }
func (s *RemoteSession) Resize(rows, cols int) error {
	_, err := fmt.Fprintf(s.Ctl, "resize %dx%d\n", cols, rows)
	return err
}
func (s *RemoteSession) Close() error {
	fmt.Fprint(s.Ctl, "kill\n")
	s.Ctl.Close()
	return s.Io.Close()
}
