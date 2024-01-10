package token

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

type FileSet struct {
	mutex sync.RWMutex
	base  int
	files []*File

	// replaced atomic.Pointer[File] for go1.19 without generics
	last atomicFilePointer
}

type atomicFilePointer struct {
	v unsafe.Pointer
}

func (x *atomicFilePointer) Load() *File     { return (*File)(atomic.LoadPointer(&x.v)) }
func (x *atomicFilePointer) Store(val *File) { atomic.StorePointer(&x.v, unsafe.Pointer(val)) }
