//go:build js

package reflect

import (
	"unsafe"

	"github.com/gopherjs/gopherjs/js"
)

//gopherjs:replace
func makeMethodValue(op string, v Value) Value {
	if v.flag&flagMethod == 0 {
		panic("reflect: internal error: invalid use of makePartialFunc")
	}

	_, _, fn := methodReceiver(op, v, int(v.flag)>>flagMethodShift)
	rcvr := v.object()
	if v.typ().IsWrapped() {
		rcvr = v.typ().JsType().New(rcvr)
	}
	fv := js.MakeFunc(func(this *js.Object, arguments []*js.Object) any {
		return js.InternalObject(fn).Call("apply", rcvr, arguments)
	})
	return Value{
		typ_: v.Type().common(),
		ptr:  unsafe.Pointer(fv.Unsafe()),
		flag: v.flag.ro() | flag(Func),
	}
}

//gopherjs:purge
func makeFuncStub()
