//go:build js
// +build js

package sync

type Map struct {
	mu Mutex

	// replaced atomic.Pointer[readOnly] since GopherJS does not fully support generics for go1.20 yet.
	read atomicReadOnlyPointer

	dirty  map[any]*entry
	misses int
}

type atomicReadOnlyPointer struct {
	v *readOnly
}

func (x *atomicReadOnlyPointer) Load() *readOnly     { return x.v }
func (x *atomicReadOnlyPointer) Store(val *readOnly) { x.v = val }
