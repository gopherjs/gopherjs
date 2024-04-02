//go:build js
// +build js

package token

import "sync"

type FileSet struct {
	mutex sync.RWMutex
	base  int
	files []*File

	// temporarily replacement of atomic.Pointer[File] for go1.20 without generics.
	last atomicFilePointer
}

type atomicFilePointer struct {
	v *File
}

func (x *atomicFilePointer) Load() *File     { return x.v }
func (x *atomicFilePointer) Store(val *File) { x.v = val }

func (x *atomicFilePointer) CompareAndSwap(old, new *File) bool {
	if x.v == old {
		x.v = new
		return true
	}
	return false
}
