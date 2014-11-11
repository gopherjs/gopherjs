// +build js

package json

import (
	"reflect"
)

type nopRWLocker struct{}

func (nopRWLocker) Lock()    {}
func (nopRWLocker) Unlock()  {}
func (nopRWLocker) RLock()   {}
func (nopRWLocker) RUnlock() {}

var fieldCache struct {
	nopRWLocker
	m map[reflect.Type][]field
}

var encoderCache struct {
	nopRWLocker
	m map[reflect.Type]encoderFunc
}

func typeEncoder(t reflect.Type) encoderFunc {
	f := encoderCache.m[t]
	if f != nil {
		return f
	}

	// To deal with recursive types, populate the map with an
	// indirect func before we build it. This type waits on the
	// real func (f) to be ready and then calls it.  This indirect
	// func is only used for recursive types.
	if encoderCache.m == nil {
		encoderCache.m = make(map[reflect.Type]encoderFunc)
	}
	encoderCache.m[t] = func(e *encodeState, v reflect.Value, quoted bool) {
		f(e, v, quoted)
	}

	f = newTypeEncoder(t, true)
	encoderCache.m[t] = f
	return f
}
