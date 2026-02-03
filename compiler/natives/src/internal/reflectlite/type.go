//go:build js

package reflectlite

import (
	"unsafe"

	"internal/abi"
)

func (t rtype) Comparable() bool {
	switch t.Kind() {
	case abi.Func, abi.Slice, abi.Map:
		return false
	case abi.Array:
		return t.Elem().Comparable()
	case abi.Struct:
		st := t.StructType()
		for i := 0; i < len(st.Fields); i++ {
			if !toRType(st.Fields[i].Typ).Comparable() {
				return false
			}
		}
	}
	return true
}

//gopherjs:purge The name type is mostly unused, replaced by abi.Name, except in pkgPath which we don't implement.
type name struct{}

func (t rtype) nameOff(off nameOff) abi.Name {
	return t.NameOff(off)
}

func (t rtype) typeOff(off typeOff) *abi.Type {
	return t.TypeOff(off)
}

func pkgPath(n abi.Name) string { return "" }

//gopherjs:purge Unused function because of nameOffList in internal/abi overrides
func resolveNameOff(ptrInModule unsafe.Pointer, off int32) unsafe.Pointer

//gopherjs:purge Unused function because of typeOffList in internal/abi overrides
func resolveTypeOff(rtype unsafe.Pointer, off int32) unsafe.Pointer
