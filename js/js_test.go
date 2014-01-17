package js_test

import (
	"github.com/neelance/gopherjs/js"
	"testing"
)

var IntVar int = 0
var ArrayVar = [3]int{41, 42, 43}

var pkg = js.Global("go$pkg")

func TestSet(t *testing.T) {
	pkg.Set("IntVar", 42)
	if IntVar != 42 {
		t.Errorf("expected %#v, got %#v", 42, IntVar)
	}
}

func TestBool(t *testing.T) {
	e := true
	pkg.Set("test", e)
	o := pkg.Get("test")
	if v := o.Bool(); v != e {
		t.Errorf("expected %#v, got %#v", e, v)
	}
	if i := o.Interface().(bool); i != e {
		t.Errorf("expected %#v, got %#v", e, i)
	}
}

func TestSting(t *testing.T) {
	e := "abc"
	pkg.Set("test", e)
	o := pkg.Get("test")
	if v := o.String(); v != e {
		t.Errorf("expected %#v, got %#v", e, v)
	}
	if i := o.Interface().(string); i != e {
		t.Errorf("expected %#v, got %#v", e, i)
	}
}

func TestInt(t *testing.T) {
	e := 42
	pkg.Set("test", e)
	o := pkg.Get("test")
	if v := o.Int(); v != e {
		t.Errorf("expected %#v, got %#v", e, v)
	}
	if i := int(o.Interface().(float64)); i != e {
		t.Errorf("expected %#v, got %#v", e, i)
	}
}

func TestFloat(t *testing.T) {
	e := 42.123
	pkg.Set("test", e)
	o := pkg.Get("test")
	if v := o.Float(); v != e {
		t.Errorf("expected %#v, got %#v", e, v)
	}
	if i := o.Interface().(float64); i != e {
		t.Errorf("expected %#v, got %#v", e, i)
	}
}

func TestIsUndefined(t *testing.T) {
	if pkg.IsUndefined() || !pkg.Get("xyz").IsUndefined() {
		t.Fail()
	}
}

func TestIsNull(t *testing.T) {
	pkg.Set("test", nil)
	if pkg.IsNull() || !pkg.Get("test").IsNull() {
		t.Fail()
	}
}

func TestLength(t *testing.T) {
	if pkg.Get("ArrayVar").Length() != 3 {
		t.Fail()
	}
}

func TestIndex(t *testing.T) {
	if pkg.Get("ArrayVar").Index(1).Int() != 42 {
		t.Fail()
	}
}

func TestSetIndex(t *testing.T) {
	pkg.Get("ArrayVar").SetIndex(2, 99)
	if pkg.Get("ArrayVar").Index(2).Int() != 99 {
		t.Fail()
	}
}

func TestCall(t *testing.T) {
	if js.Global("go$global").Call("parseInt", "42").Interface().(float64) != 42 {
		t.Fail()
	}
}

func TestInvoke(t *testing.T) {
	if js.Global("parseInt").Invoke("42").Interface().(float64) != 42 {
		t.Fail()
	}
}

func TestNew(t *testing.T) {
	if js.Global("Array").New(42).Length() != 42 {
		t.Fail()
	}
}
