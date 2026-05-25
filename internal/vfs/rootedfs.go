package vfs

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/aleksana/peak/internal/vfs/afero"
)

// rootedFs exposes a subtree of a CompositeFs to the 9P server. It wraps
// BasePathFs for path translation and propagates WalkRedirector calls through
// the composite to the appropriate submount.
type rootedFs struct {
	afero.Fs
	composite *CompositeFs
	base      string
}

// NewRootedFs returns an afero.Fs that exposes the composite subtree at base.
// The returned filesystem translates server-relative paths to composite-absolute
// paths and implements WalkRedirector by delegating to the composite.
func NewRootedFs(composite *CompositeFs, base string) afero.Fs {
	return &rootedFs{
		Fs:        afero.NewBasePathFs(composite, base),
		composite: composite,
		base:      filepath.Clean(base),
	}
}

func (f *rootedFs) WalkRedirect(dir, name string) (string, os.FileInfo, bool) {
	// dir is server-relative (e.g. "/"); convert to composite-absolute.
	absDir := filepath.Join(f.base, filepath.Clean(dir))
	rp, fi, ok := f.composite.WalkRedirect(absDir, name)
	if !ok {
		return "", nil, false
	}
	// rp is composite-absolute; strip base to get server-relative path.
	rel := strings.TrimPrefix(filepath.Clean(rp), f.base)
	if rel == "" {
		rel = "/"
	}
	return rel, fi, true
}
