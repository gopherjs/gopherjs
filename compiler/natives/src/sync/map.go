//go:build js
// +build js

package sync

type Map struct {
	mu Mutex

	// replaced atomic.Pointer[readOnly] for go1.20 without generics.
	read atomicReadOnlyPointer

	dirty  map[any]*entry
	misses int
}

type atomicReadOnlyPointer struct {
	v *readOnly
}

func (x *atomicReadOnlyPointer) Load() *readOnly     { return x.v }
func (x *atomicReadOnlyPointer) Store(val *readOnly) { x.v = val }

type entry struct {

	// replaced atomic.Pointer[any] for go1.20 without generics.
	p atomicAnyPointer
}

type atomicAnyPointer struct {
	v *any
}

func (x *atomicAnyPointer) Load() *any     { return x.v }
func (x *atomicAnyPointer) Store(val *any) { x.v = val }

func (x *atomicAnyPointer) Swap(new *any) *any {
	old := x.v
	x.v = new
	return old
}

func (x *atomicAnyPointer) CompareAndSwap(old, new *any) bool {
	if x.v == old {
		x.v = new
		return true
	}
	return false
}
