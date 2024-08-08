//go:build js
// +build js

package gob

import "sync"

type typeInfo struct {
	id      typeId
	encInit sync.Mutex

	// replacing a `atomic.Pointer[encEngine]` since GopherJS does not fully support generics for go1.20 yet.
	encoder atomicEncEnginePointer
	wire    *wireType
}

type atomicEncEnginePointer struct {
	v *encEngine
}

func (x *atomicEncEnginePointer) Load() *encEngine     { return x.v }
func (x *atomicEncEnginePointer) Store(val *encEngine) { x.v = val }
