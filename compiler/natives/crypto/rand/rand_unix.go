// +build js

package rand

import (
	"io"
)

// A devReader satisfies reads by reading the file named name.
type devReader struct {
	name string
	f    io.Reader
	mu   nopLocker
}

type nopLocker struct{}

func (nopLocker) Lock()   {}
func (nopLocker) Unlock() {}
