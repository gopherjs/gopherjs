//go:build js

package reflectlite

import (
	"internal/abi"
)

// GOPHERJS: For some reason the original code left mapType and aliased the rest
// to the ABI version. mapType is not used so this is an alias to override the
// left over refactor cruft.
//
//gopherjs:replace
type mapType = abi.MapType

//gopherjs:replace
func (t rtype) Comparable() bool {
	switch t.Kind() {
	case abi.Func, abi.Slice, abi.Map:
		return false
	case abi.Array:
		return t.Elem().Comparable()
	case abi.Struct:
		st := t.StructType()
		for i := 0; i < len(st.Fields); i++ {
			ft := st.Fields[i]
			if !toRType(ft.Typ).Comparable() {
				return false
			}
		}
	}
	return true
}
