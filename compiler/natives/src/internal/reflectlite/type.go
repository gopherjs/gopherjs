//go:build js

package reflectlite

import (
	"internal/abi"

	"github.com/gopherjs/gopherjs/js"
)

var nameOffList []abi.Name

func (t rtype) nameOff(off nameOff) abi.Name {
	return nameOffList[int(off)]
}

func newNameOff(n abi.Name) nameOff {
	i := len(nameOffList)
	nameOffList = append(nameOffList, n)
	return nameOff(i)
}

var typeOffList []*abi.Type

func (t rtype) typeOff(off typeOff) *abi.Type {
	return typeOffList[int(off)]
}

func newTypeOff(t *abi.Type) typeOff {
	i := len(typeOffList)
	typeOffList = append(typeOffList, t)
	return typeOff(i)
}

func (t rtype) ptrTo() rtype {
	return reflectType(js.Global.Call("$ptrType", jsType(t.Type)))
}

func (t rtype) Comparable() bool {
	switch t.Kind() {
	case abi.Func, abi.Slice, abi.Map:
		return false
	case abi.Array:
		return t.Elem().Comparable()
	case abi.Struct:
		for i := 0; i < t.NumField(); i++ {
			ft := t.Field(i)
			if !toRType(ft.Typ).Comparable() {
				return false
			}
		}
	}
	return true
}

//gopherjs:purge The name type is mostly unused, replaced by abi.Name, except in pkgPath which we don't implement.
type name struct{}

func pkgPath(n abi.Name) string { return "" }
