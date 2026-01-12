//go:build js

package net

import (
	"os"
	_ "unsafe" // for go:linkname
)

// Reversing the linkname direction
//
//go:linkname newUnixFile os.net_newUnixFile
func newUnixFile(fd uintptr, name string) *os.File
