// +build js
// +build go1.14

package js

import (
	"github.com/gopherjs/gopherjs/js"
)

func (v Value) IsNull() bool {
	return v.Type() == TypeNull
}

func (v Value) IsUndefined() bool {
	return !v.inited
}

func (v Value) IsNaN() bool {
	return js.Global.Call("isNaN", v.internal()).Bool()
}

func (v Value) Equal(w Value) bool {
	//return v.internal() == w.internal() && !v.IsNaN()
	if v.Type() != w.Type() {
		return false
	}
	switch v.Type() {
	case TypeString:
		return v.internal().String() == w.internal().String()
	case TypeUndefined:
		return true
	case TypeNull:
		return true
	case TypeBoolean:
		return v.internal().Bool() == w.internal().Bool()
	case TypeNumber:
		return v.internal().Float() == w.internal().Float()
	case TypeSymbol:
		return v.internal() == w.internal()
	case TypeObject:
		return v.internal() == w.internal()
	case TypeFunction:
		return v.internal() == w.internal()
	default:
		panic("bad type")
	}
}

func (v Value) Delete(p string) {
	if vType := v.Type(); vType != TypeObject {
		panic(&ValueError{"Value.Delete", vType})
	}
	v.internal().Delete(p)
}
