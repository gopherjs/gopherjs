package js_test

import (
	"github.com/neelance/gopherjs/js"
	"testing"
	"time"
)

var dummys js.Object

func TestBool(t *testing.T) {
	e := true
	o := dummys.Get("someBool")
	if v := o.Bool(); v != e {
		t.Errorf("expected %#v, got %#v", e, v)
	}
	if i := o.Interface().(bool); i != e {
		t.Errorf("expected %#v, got %#v", e, i)
	}
	if dummys.Set("otherBool", e); dummys.Get("otherBool").Bool() != e {
		t.Fail()
	}
}

func TestString(t *testing.T) {
	e := "abc\u1234"
	o := dummys.Get("someString")
	if v := o.String(); v != e {
		t.Errorf("expected %#v, got %#v", e, v)
	}
	if i := o.Interface().(string); i != e {
		t.Errorf("expected %#v, got %#v", e, i)
	}
	if dummys.Set("otherString", e); dummys.Get("otherString").String() != e {
		t.Fail()
	}
}

func TestInt(t *testing.T) {
	e := 42
	o := dummys.Get("someInt")
	if v := o.Int(); v != e {
		t.Errorf("expected %#v, got %#v", e, v)
	}
	if i := int(o.Interface().(float64)); i != e {
		t.Errorf("expected %#v, got %#v", e, i)
	}
	if dummys.Set("otherInt", e); dummys.Get("otherInt").Int() != e {
		t.Fail()
	}
}

func TestFloat(t *testing.T) {
	e := 42.123
	o := dummys.Get("someFloat")
	if v := o.Float(); v != e {
		t.Errorf("expected %#v, got %#v", e, v)
	}
	if i := o.Interface().(float64); i != e {
		t.Errorf("expected %#v, got %#v", e, i)
	}
	if dummys.Set("otherFloat", e); dummys.Get("otherFloat").Float() != e {
		t.Fail()
	}
}

func TestIsUndefined(t *testing.T) {
	if dummys.IsUndefined() || !dummys.Get("xyz").IsUndefined() {
		t.Fail()
	}
}

func TestIsNull(t *testing.T) {
	dummys.Set("test", nil)
	if dummys.IsNull() || !dummys.Get("test").IsNull() {
		t.Fail()
	}
}

func TestLength(t *testing.T) {
	if dummys.Get("someArray").Length() != 3 {
		t.Fail()
	}
}

func TestIndex(t *testing.T) {
	if dummys.Get("someArray").Index(1).Int() != 42 {
		t.Fail()
	}
}

func TestSetIndex(t *testing.T) {
	dummys.Get("someArray").SetIndex(2, 99)
	if dummys.Get("someArray").Index(2).Int() != 99 {
		t.Fail()
	}
}

func TestCall(t *testing.T) {
	var i int64 = 40
	if dummys.Call("add", i, 2).Int() != 42 {
		t.Fail()
	}
}

func TestInvoke(t *testing.T) {
	var i int64 = 40
	if dummys.Get("add").Invoke(i, 2).Int() != 42 {
		t.Fail()
	}
}

func TestNew(t *testing.T) {
	if js.Global("Array").New(42).Length() != 42 {
		t.Fail()
	}
}

type StructWithJsField struct {
	js.Object
	Length int                  `js:"length"`
	Slice  func(int, int) []int `js:"slice"`
}

type StructWithJsField2 struct {
	object js.Object            // to hide members from public API
	Length int                  `js:"length"`
	Slice  func(int, int) []int `js:"slice"`
}

func TestReadingJsField(t *testing.T) {
	a := &StructWithJsField{Object: js.Global("Array").New(42)}
	b := &StructWithJsField2{object: js.Global("Array").New(42)}
	if a.Length != 42 || b.Length != 42 {
		t.Fail()
	}
}

func TestWritingJsField(t *testing.T) {
	a := &StructWithJsField{Object: js.Global("Object").New()}
	b := &StructWithJsField2{object: js.Global("Object").New()}
	a.Length = 42
	b.Length = 42
	if a.Get("length").Int() != 42 || b.object.Get("length").Int() != 42 {
		t.Fail()
	}
}

func TestCallingJsField(t *testing.T) {
	a := &StructWithJsField{Object: js.Global("Array").New(100)}
	b := &StructWithJsField2{object: js.Global("Array").New(100)}
	a.SetIndex(3, 123)
	b.object.SetIndex(3, 123)
	f := a.Slice
	a2 := a.Slice(2, 44)
	b2 := b.Slice(2, 44)
	c2 := f(2, 44)
	if len(a2) != 42 || len(b2) != 42 || len(c2) != 42 || a2[1] != 123 || b2[1] != 123 || c2[1] != 123 {
		t.Fail()
	}
}

func TestFunc(t *testing.T) {
	a := dummys.Call("mapArray", []int{1, 2, 3}, func(e int64) int64 { return e + 40 })
	b := dummys.Call("mapArray", []int{1, 2, 3}, func(e ...int64) int64 { return e[0] + 40 })
	if a.Index(1).Int() != 42 || b.Index(1).Int() != 42 {
		t.Fail()
	}

	add := dummys.Get("add").Interface().(func(...interface{}) js.Object)
	var i int64 = 40
	if add(i, 2).Int() != 42 {
		t.Fail()
	}
}

func TestDate(t *testing.T) {
	d := time.Date(2013, time.August, 27, 22, 25, 11, 0, time.UTC)
	if dummys.Call("toUnixTimestamp", d).Int() != int(d.Unix()) {
		t.Fail()
	}

	d2 := js.Global("Date").New(d.UnixNano() / 1000000).Interface().(time.Time)
	if !d2.Equal(d) {
		t.Fail()
	}
}

func TestEquality(t *testing.T) {
	if js.Global("Array") != js.Global("Array") || js.Global("Array") == js.Global("String") {
		t.Fail()
	}
}

func TestThis(t *testing.T) {
	dummys.Set("testThis", func() {
		if js.This() != dummys {
			t.Fail()
		}
	})
	dummys.Call("testThis")
}
