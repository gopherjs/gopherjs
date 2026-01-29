//go:build js

package reflectlite

import (
	"unsafe"

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

func (t *rtype) typeOff(off typeOff) *abi.Type {
	return typeOffList[int(off)]
}

func newTypeOff(t *rtype) typeOff {
	i := len(typeOffList)
	typeOffList = append(typeOffList, t)
	return typeOff(i)
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
			if !ft.typ.Comparable() {
				return false
			}
		}
	}
	return true
}

func (t *rtype) IsVariadic() bool {
	if t.Kind() != abi.Func {
		panic("reflect: IsVariadic of non-func type")
	}
	tt := (*funcType)(unsafe.Pointer(t))
	return tt.outCount&(1<<15) != 0
}

func (t *rtype) kindType() *rtype {
	return (*rtype)(unsafe.Pointer(js.InternalObject(t).Get(idKindType)))
}

func (t *rtype) Key() Type {
	if t.Kind() != abi.Map {
		panic("reflect: Key of non-map type")
	}
	tt := (*mapType)(unsafe.Pointer(t))
	return toType(tt.key)
}

func (t *rtype) NumField() int {
	if t.Kind() != abi.Struct {
		panic("reflect: NumField of non-struct type")
	}
	tt := (*structType)(unsafe.Pointer(t))
	return len(tt.fields)
}

func (t *rtype) Method(i int) (m Method) {
	if t.Kind() == abi.Interface {
		tt := (*interfaceType)(unsafe.Pointer(t))
		return tt.Method(i)
	}
	methods := t.exportedMethods()
	if i < 0 || i >= len(methods) {
		panic("reflect: Method index out of range")
	}
	p := methods[i]
	pname := t.nameOff(p.name)
	m.Name = pname.name()
	fl := flag(Func)
	mtyp := t.typeOff(p.mtyp)
	ft := (*funcType)(unsafe.Pointer(mtyp))
	in := make([]Type, 0, 1+len(ft.in()))
	in = append(in, t)
	for _, arg := range ft.in() {
		in = append(in, arg)
	}
	out := make([]Type, 0, len(ft.out()))
	for _, ret := range ft.out() {
		out = append(out, ret)
	}
	mt := FuncOf(in, out, ft.IsVariadic())
	m.Type = mt
	prop := js.Global.Call("$methodSet", js.InternalObject(t).Get(idJsType)).Index(i).Get("prop").String()
	fn := js.MakeFunc(func(this *js.Object, arguments []*js.Object) any {
		rcvr := arguments[0]
		return rcvr.Get(prop).Call("apply", rcvr, arguments[1:])
	})
	m.Func = Value{mt.(*rtype), unsafe.Pointer(fn.Unsafe()), fl}

	m.Index = i
	return m
}
