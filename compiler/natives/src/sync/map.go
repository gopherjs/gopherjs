//go:build js
// +build js

package sync

type Map struct {
	mu Mutex

	// TODO(grantnelson-wf): Remove this override after generics are supported.
	// https://github.com/gopherjs/gopherjs/issues/1013.
	//
	// This override is still needed with initial generics support because otherwise we get:
	//	[compiler panic]  unexpected compiler panic while building package "reflect":
	//	requesting ID of instance {type sync/atomic.Pointer[T any] struct{_ [0]*T; _ sync/atomic.noCopy; v unsafe.Pointer} sync.readOnly}
	//	that hasn't been added to the set
	read atomicReadOnlyPointer

	dirty  map[any]*entry
	misses int
}

type atomicReadOnlyPointer struct {
	v *readOnly
}

func (x *atomicReadOnlyPointer) Load() *readOnly     { return x.v }
func (x *atomicReadOnlyPointer) Store(val *readOnly) { x.v = val }
