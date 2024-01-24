//go:build js
// +build js

package token

import (
	"sync"
)

type FileSet struct {
	mutex sync.RWMutex
	base  int
	files []*File

	// replaced atomic.Pointer[File] for go1.19 without generics.
	last atomicFilePointer
}

type atomicFilePointer struct {
	v *File
}

func (x *atomicFilePointer) Load() *File     { return x.v }
func (x *atomicFilePointer) Store(val *File) { x.v = val }
