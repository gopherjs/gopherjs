//go:build js

package reflectlite

import (
	"unsafe"

	"internal/abi"

	"github.com/gopherjs/gopherjs/js"
)

// GOPHERJS: For some reason the original code left mapType and aliased the rest
// to the ABI version. mapType is not used so this is an alias to override the
// left over refactor cruft.
//
//gopherjs:replace
type mapType = abi.MapType

//gopherjs:purge The name type is mostly unused, replaced by abi.Name, except in pkgPath which we don't implement.
type name struct{}

//gopherjs:replace
func pkgPath(n abi.Name) string { return "" }

//gopherjs:purge Unused function because of nameOffList in internal/abi overrides
func resolveNameOff(ptrInModule unsafe.Pointer, off int32) unsafe.Pointer

//gopherjs:purge Unused function because of typeOffList in internal/abi overrides
func resolveTypeOff(rtype unsafe.Pointer, off int32) unsafe.Pointer

//gopherjs:replace
func (t rtype) nameOff(off nameOff) abi.Name {
	return t.NameOff(off)
}

//gopherjs:replace
func (t rtype) typeOff(off typeOff) *abi.Type {
	return t.TypeOff(off)
}

//gopherjs:replace
func TypeOf(i any) Type {
	if i == nil {
		return nil
	}
	return toRType(abi.ReflectType(js.InternalObject(i).Get("constructor")))
}

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
			if !toRType(st.Fields[i].Typ).Comparable() {
				return false
			}
		}
	}
	return true
}
