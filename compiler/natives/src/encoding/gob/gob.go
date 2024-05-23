//go:build js
// +build js

package gob

import (
	"reflect"
	"sync"
)

type typeInfo struct {
	id      typeId
	encInit sync.Mutex

	// temporarily replacement of atomic.Pointer[encEngine] for go1.20 without generics.
	encoder atomicEncEnginePointer
	wire    *wireType
}

type atomicEncEnginePointer struct {
	v *encEngine
}

func (x *atomicEncEnginePointer) Load() *encEngine     { return x.v }
func (x *atomicEncEnginePointer) Store(val *encEngine) { x.v = val }

// temporarily replacement of growSlice[E any] for go1.20 without generics.
func growSlice(v reflect.Value, ps any, length int) {
	vps := reflect.ValueOf(ps)
	vs := vps.Elem()
	zero := reflect.Zero(vs.Elem().Type())
	vs.Set(reflect.Append(vs, zero))
	cp := vs.Cap()
	if cp > length {
		cp = length
	}
	vs.Set(vs.Slice(0, cp))
	v.Set(vs)
	vps.Set(vs.Addr())
}
