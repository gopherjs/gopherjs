//go:build js
// +build js

package js

import (
	"reflect"
	"unsafe"

	"github.com/gopherjs/gopherjs/js"
)

type Type int

const (
	TypeUndefined Type = iota
	TypeNull
	TypeBoolean
	TypeNumber
	TypeString
	TypeSymbol
	TypeObject
	TypeFunction
)

func (t Type) String() string {
	switch t {
	case TypeUndefined:
		return "undefined"
	case TypeNull:
		return "null"
	case TypeBoolean:
		return "boolean"
	case TypeNumber:
		return "number"
	case TypeString:
		return "string"
	case TypeSymbol:
		return "symbol"
	case TypeObject:
		return "object"
	case TypeFunction:
		return "function"
	default:
		panic("bad type")
	}
}

func (t Type) isObject() bool {
	return t == TypeObject || t == TypeFunction
}

func Global() Value {
	return objectToValue(js.Global)
}

func Null() Value {
	return objectToValue(nil)
}

func Undefined() Value {
	return objectToValue(js.Undefined)
}

type Func struct {
	Value
}

func (f Func) Release() {
	js.Global.Set("$exportedFunctions", js.Global.Get("$exportedFunctions").Int()-1)
	f.Value = Null()
}

func FuncOf(fn func(this Value, args []Value) interface{}) Func {
	// Existence of a wrapped function means that an external event may awaken the
	// program and we need to suppress deadlock detection.
	js.Global.Set("$exportedFunctions", js.Global.Get("$exportedFunctions").Int()+1)
	return Func{
		Value: objectToValue(js.MakeFunc(func(this *js.Object, args []*js.Object) interface{} {
			vargs := make([]Value, len(args))
			for i, a := range args {
				vargs[i] = objectToValue(a)
			}
			return fn(objectToValue(this), vargs)
		})),
	}
}

type Error struct {
	Value
}

func (e Error) Error() string {
	return "JavaScript error: " + e.Get("message").String()
}

type Value struct {
	v *js.Object

	// inited represents whether Value is non-zero value. true represents the value is not 'undefined'.
	inited bool

	_ [0]func() // uncomparable; to make == not compile
}

func objectToValue(obj *js.Object) Value {
	if obj == js.Undefined {
		return Value{}
	}
	return Value{v: obj, inited: true}
}

var (
	id           *js.Object
	instanceOf   *js.Object
	getValueType *js.Object
)

func init() {
	if js.Global != nil {
		id = js.Global.Call("eval", "(function(x) { return x; })")
		instanceOf = js.Global.Call("eval", "(function(x, y) { return x instanceof y; })")
		getValueType = js.Global.Call("eval", `(function(x) {
  if (typeof(x) === "undefined") {
    return 0; // TypeUndefined
  }
  if (x === null) {
    return 1; // TypeNull
  }
  if (typeof(x) === "boolean") {
    return 2; // TypeBoolean
  }
  if (typeof(x) === "number") {
    return 3; // TypeNumber
  }
  if (typeof(x) === "string") {
    return 4; // TypeString
  }
  if (typeof(x) === "symbol") {
    return 5; // TypeSymbol
  }
  if (typeof(x) === "function") {
    return 7; // TypeFunction
  }
  return 6; // TypeObject
})`)
	}
}

func ValueOf(x interface{}) Value {
	switch x := x.(type) {
	case Wrapper:
		return x.JSValue()
	case Value:
		return x
	case Func:
		return x.Value
	case nil:
		return Null()
	case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, unsafe.Pointer, string, map[string]interface{}, []interface{}:
		return objectToValue(id.Invoke(x))
	default:
		panic(`invalid arg: ` + reflect.TypeOf(x).String())
	}
}

func (v Value) internal() *js.Object {
	if !v.inited {
		return js.Undefined
	}
	return v.v
}

func (v Value) Bool() bool {
	if vType := v.Type(); vType != TypeBoolean {
		panic(&ValueError{"Value.Bool", vType})
	}
	return v.internal().Bool()
}

// convertArgs converts arguments into values for GopherJS arguments.
func convertArgs(args ...interface{}) []interface{} {
	newArgs := []interface{}{}
	for _, arg := range args {
		v := ValueOf(arg)
		newArgs = append(newArgs, v.internal())
	}
	return newArgs
}

func (v Value) Call(m string, args ...interface{}) Value {
	if vType := v.Type(); vType != TypeObject && vType != TypeFunction {
		panic(&ValueError{"Value.Call", vType})
	}
	if propType := v.Get(m).Type(); propType != TypeFunction {
		panic("js: Value.Call: property " + m + " is not a function, got " + propType.String())
	}
	return objectToValue(v.internal().Call(m, convertArgs(args...)...))
}

func (v Value) Float() float64 {
	if vType := v.Type(); vType != TypeNumber {
		panic(&ValueError{"Value.Float", vType})
	}
	return v.internal().Float()
}

func (v Value) Get(p string) Value {
	if vType := v.Type(); !vType.isObject() {
		panic(&ValueError{"Value.Get", vType})
	}
	return objectToValue(v.internal().Get(p))
}

func (v Value) Index(i int) Value {
	if vType := v.Type(); !vType.isObject() {
		panic(&ValueError{"Value.Index", vType})
	}
	return objectToValue(v.internal().Index(i))
}

func (v Value) Int() int {
	if vType := v.Type(); vType != TypeNumber {
		panic(&ValueError{"Value.Int", vType})
	}
	return v.internal().Int()
}

func (v Value) InstanceOf(t Value) bool {
	return instanceOf.Invoke(v.internal(), t.internal()).Bool()
}

func (v Value) Invoke(args ...interface{}) Value {
	if vType := v.Type(); vType != TypeFunction {
		panic(&ValueError{"Value.Invoke", vType})
	}
	return objectToValue(v.internal().Invoke(convertArgs(args...)...))
}

func (v Value) JSValue() Value {
	return v
}

func (v Value) Length() int {
	return v.internal().Length()
}

func (v Value) New(args ...interface{}) Value {
	defer func() {
		err := recover()
		if err == nil {
			return
		}
		if vType := v.Type(); vType != TypeFunction { // check here to avoid overhead in success case
			panic(&ValueError{"Value.New", vType})
		}
		if jsErr, ok := err.(*js.Error); ok {
			panic(Error{objectToValue(jsErr.Object)})
		}
		panic(err)
	}()
	return objectToValue(v.internal().New(convertArgs(args...)...))
}

func (v Value) Set(p string, x interface{}) {
	if vType := v.Type(); !vType.isObject() {
		panic(&ValueError{"Value.Set", vType})
	}
	v.internal().Set(p, convertArgs(x)[0])
}

func (v Value) SetIndex(i int, x interface{}) {
	if vType := v.Type(); !vType.isObject() {
		panic(&ValueError{"Value.SetIndex", vType})
	}
	v.internal().SetIndex(i, convertArgs(x)[0])
}

// String returns the value v as a string.
// String is a special case because of Go's String method convention. Unlike the other getters,
// it does not panic if v's Type is not TypeString. Instead, it returns a string of the form "<T>"
// or "<T: V>" where T is v's type and V is a string representation of v's value.
func (v Value) String() string {
	switch v.Type() {
	case TypeString:
		return v.internal().String()
	case TypeUndefined:
		return "<undefined>"
	case TypeNull:
		return "<null>"
	case TypeBoolean:
		return "<boolean: " + v.internal().String() + ">"
	case TypeNumber:
		return "<number: " + v.internal().String() + ">"
	case TypeSymbol:
		return "<symbol>"
	case TypeObject:
		return "<object>"
	case TypeFunction:
		return "<function>"
	default:
		panic("bad type")
	}
}

func (v Value) Truthy() bool {
	return v.internal().Bool()
}

func (v Value) Type() Type {
	return Type(getValueType.Invoke(v.internal()).Int())
}

func (v Value) IsNull() bool {
	return v.Type() == TypeNull
}

func (v Value) IsUndefined() bool {
	return !v.inited
}

func (v Value) IsNaN() bool {
	return js.Global.Call("isNaN", v.internal()).Bool()
}

// Delete deletes the JavaScript property p of value v.
// It panics if v is not a JavaScript object.
func (v Value) Delete(p string) {
	if vType := v.Type(); !vType.isObject() {
		panic(&ValueError{"Value.Delete", vType})
	}
	v.internal().Delete(p)
}

func (v Value) Equal(w Value) bool {
	return v.internal() == w.internal()
}

type ValueError struct {
	Method string
	Type   Type
}

func (e *ValueError) Error() string {
	return "syscall/js: call of " + e.Method + " on " + e.Type.String()
}

type Wrapper interface {
	JSValue() Value
}

// CopyBytesToGo copies bytes from the Uint8Array src to dst.
// It returns the number of bytes copied, which will be the minimum of the lengths of src and dst.
// CopyBytesToGo panics if src is not an Uint8Array.
func CopyBytesToGo(dst []byte, src Value) int {
	vlen := src.v.Length()
	if dlen := len(dst); dlen < vlen {
		vlen = dlen
	}
	copy(dst, src.v.Interface().([]byte))
	return vlen
}

// CopyBytesToJS copies bytes from src to the Uint8Array dst.
// It returns the number of bytes copied, which will be the minimum of the lengths of src and dst.
// CopyBytesToJS panics if dst is not an Uint8Array.
func CopyBytesToJS(dst Value, src []byte) int {
	dt, ok := dst.v.Interface().([]byte)
	if !ok {
		panic("syscall/js: CopyBytesToJS: expected dst to be an Uint8Array")
	}
	return copy(dt, src)
}
