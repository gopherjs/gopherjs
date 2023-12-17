package typesutil

import (
	"go/types"

	"golang.org/x/tools/go/types/typeutil"
)

// Map is a type-safe wrapper around golang.org/x/tools/go/types/typeutil.Map.
type Map[Val any] struct{ impl typeutil.Map }

func (m *Map[Val]) At(key types.Type) Val {
	val := m.impl.At(key)
	if val != nil {
		return val.(Val)
	}
	var zero Val
	return zero
}

func (m *Map[Val]) Set(key types.Type, value Val) (prev Val) {
	old := m.impl.Set(key, value)
	if old != nil {
		return old.(Val)
	}
	var zero Val
	return zero
}

func (m *Map[Val]) Delete(key types.Type) bool { return m.impl.Delete(key) }

func (m *Map[Val]) Len() int { return m.impl.Len() }

func (m *Map[Val]) String() string { return m.impl.String() }
