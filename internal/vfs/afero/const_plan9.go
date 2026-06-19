//go:build plan9

package afero

import "errors"

var BADFD = errors.New("bad file descriptor")
