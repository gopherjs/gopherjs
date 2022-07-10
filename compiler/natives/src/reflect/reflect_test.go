//go:build js
// +build js

package reflect_test

import (
	"math"
	. "reflect"
	"testing"
)

func TestAlignment(t *testing.T) {
	t.Skip()
}

func TestSliceOverflow(t *testing.T) {
	t.Skip()
}

func TestFuncLayout(t *testing.T) {
	t.Skip()
}

func TestArrayOfDirectIface(t *testing.T) {
	t.Skip()
}

func TestTypelinksSorted(t *testing.T) {
	t.Skip()
}

func TestGCBits(t *testing.T) {
	t.Skip()
}

func TestChanAlloc(t *testing.T) {
	t.Skip()
}

func TestNameBytesAreAligned(t *testing.T) {
	t.Skip()
}

func TestOffsetLock(t *testing.T) {
	t.Skip()
}

func TestSelectOnInvalid(t *testing.T) {
	Select([]SelectCase{
		{
			Dir:  SelectRecv,
			Chan: Value{},
		}, {
			Dir:  SelectSend,
			Chan: Value{},
			Send: ValueOf(1),
		}, {
			Dir: SelectDefault,
		},
	})
}

func TestStructOfDirectIface(t *testing.T) {
	t.Skip("reflect.Value.InterfaceData is not supported by GopherJS.")
}

func TestStructOfWithInterface(t *testing.T) {
	// TODO(nevkontakte) Most of this test actually passes, but there is something
	// about embedding fields with methods that can or can't be stored in an
	// interface value directly that GopherJS does differently from upstream. As
	// a result, GopherJS's implementation of StructOf() doesn't panic where
	// upstream does. It seems to be a result of our implementation not propagating
	// the kindDirectIface flag in struct types created by StructOf(), but at this
	// point I wasn't able to figure out what that flag actually means in the
	// GopherJS context or how it maps onto our own reflection implementation.
	t.Skip("GopherJS doesn't support storing types directly in interfaces.")
}

var deepEqualTests = []DeepEqualTest{
	// Equalities
	{nil, nil, true},
	{1, 1, true},
	{int32(1), int32(1), true},
	{0.5, 0.5, true},
	{float32(0.5), float32(0.5), true},
	{"hello", "hello", true},
	{make([]int, 10), make([]int, 10), true},
	{&[3]int{1, 2, 3}, &[3]int{1, 2, 3}, true},
	{Basic{1, 0.5}, Basic{1, 0.5}, true},
	{error(nil), error(nil), true},
	{map[int]string{1: "one", 2: "two"}, map[int]string{2: "two", 1: "one"}, true},
	{fn1, fn2, true},

	// Inequalities
	{1, 2, false},
	{int32(1), int32(2), false},
	{0.5, 0.6, false},
	{float32(0.5), float32(0.6), false},
	{"hello", "hey", false},
	{make([]int, 10), make([]int, 11), false},
	{&[3]int{1, 2, 3}, &[3]int{1, 2, 4}, false},
	{Basic{1, 0.5}, Basic{1, 0.6}, false},
	{Basic{1, 0}, Basic{2, 0}, false},
	{map[int]string{1: "one", 3: "two"}, map[int]string{2: "two", 1: "one"}, false},
	{map[int]string{1: "one", 2: "txo"}, map[int]string{2: "two", 1: "one"}, false},
	{map[int]string{1: "one"}, map[int]string{2: "two", 1: "one"}, false},
	{map[int]string{2: "two", 1: "one"}, map[int]string{1: "one"}, false},
	{nil, 1, false},
	{1, nil, false},
	{fn1, fn3, false},
	{fn3, fn3, false},
	{[][]int{{1}}, [][]int{{2}}, false},
	{math.NaN(), math.NaN(), false},
	{&[1]float64{math.NaN()}, &[1]float64{math.NaN()}, false},
	{&[1]float64{math.NaN()}, self{}, true},
	{[]float64{math.NaN()}, []float64{math.NaN()}, false},
	{[]float64{math.NaN()}, self{}, true},
	{map[float64]float64{math.NaN(): 1}, map[float64]float64{1: 2}, false},
	{map[float64]float64{math.NaN(): 1}, self{}, true},

	// Nil vs empty: not the same.
	{[]int{}, []int(nil), false},
	{[]int{}, []int{}, true},
	{[]int(nil), []int(nil), true},
	{map[int]int{}, map[int]int(nil), false},
	{map[int]int{}, map[int]int{}, true},
	{map[int]int(nil), map[int]int(nil), true},

	// Mismatched types
	{1, 1.0, false},
	{int32(1), int64(1), false},
	{0.5, "hello", false},
	{[]int{1, 2, 3}, [3]int{1, 2, 3}, false},
	{&[3]interface{}{1, 2, 4}, &[3]interface{}{1, 2, "s"}, false},
	{Basic{1, 0.5}, NotBasic{1, 0.5}, false},
	{map[uint]string{1: "one", 2: "two"}, map[int]string{2: "two", 1: "one"}, false},

	// Possible loops.
	{&loop1, &loop1, true},
	//{&loop1, &loop2, true}, // TODO: Fix.
	{&loopy1, &loopy1, true},
	//{&loopy1, &loopy2, true}, // TODO: Fix.
}

// TODO: Fix this. See https://github.com/gopherjs/gopherjs/issues/763.
func TestIssue22073(t *testing.T) {
	m := ValueOf(NonExportedFirst(0)).Method(0)

	if got := m.Type().NumOut(); got != 0 {
		t.Errorf("NumOut: got %v, want 0", got)
	}

	// TODO: Fix this. The call below fails with:
	//
	// 	var $call = function(fn, rcvr, args) { return fn.apply(rcvr, args); };
	// 	                                                 ^
	// 	TypeError: Cannot read property 'apply' of undefined

	// Shouldn't panic.
	//m.Call(nil)
}

func TestCallReturnsEmpty(t *testing.T) {
	t.Skip("test uses runtime.SetFinalizer, which is not supported by GopherJS")
}

func init() {
	// TODO: This is a failure in 1.11, try to determine the cause and fix.
	typeTests = append(typeTests[:31], typeTests[32:]...) // skip test case #31
}

func TestConvertNaNs(t *testing.T) {
	// This test is exactly the same as the upstream, except it uses a "quiet NaN"
	// value instead of "signalling NaN". JavaScript appears to coerce all NaNs
	// into quiet ones, but for the purpose of this test either is fine.

	const qnan uint32 = 0x7fc00001 // Originally: 0x7f800001.
	type myFloat32 float32
	x := V(myFloat32(math.Float32frombits(qnan)))
	y := x.Convert(TypeOf(float32(0)))
	z := y.Interface().(float32)
	if got := math.Float32bits(z); got != qnan {
		t.Errorf("quiet nan conversion got %x, want %x", got, qnan)
	}
}

func TestMapIterSet(t *testing.T) {
	m := make(map[string]any, len(valueTests))
	for _, tt := range valueTests {
		m[tt.s] = tt.i
	}
	v := ValueOf(m)

	k := New(v.Type().Key()).Elem()
	e := New(v.Type().Elem()).Elem()

	iter := v.MapRange()
	for iter.Next() {
		k.SetIterKey(iter)
		e.SetIterValue(iter)
		want := m[k.String()]
		got := e.Interface()
		if got != want {
			t.Errorf("%q: want (%T) %v, got (%T) %v", k.String(), want, want, got, got)
		}
		if setkey, key := valueToString(k), valueToString(iter.Key()); setkey != key {
			t.Errorf("MapIter.Key() = %q, MapIter.SetKey() = %q", key, setkey)
		}
		if setval, val := valueToString(e), valueToString(iter.Value()); setval != val {
			t.Errorf("MapIter.Value() = %q, MapIter.SetValue() = %q", val, setval)
		}
	}

	// Upstream test also tests allocations made by the iterator. GopherJS doesn't
	// support runtime.ReadMemStats(), so we leave that part out.
}

type inner struct {
	x int
}

type outer struct {
	y int
	inner
}

func (*inner) M() int { return 1 }
func (*outer) M() int { return 2 }

func TestNestedMethods(t *testing.T) {
	// This test is similar to the upstream, but avoids using the unsupported
	// Value.UnsafePointer() method.
	typ := TypeOf((*outer)(nil))
	args := []Value{
		ValueOf((*outer)(nil)), // nil receiver
	}
	if typ.NumMethod() != 1 {
		t.Errorf("Wrong method table for outer, found methods:")
		for i := 0; i < typ.NumMethod(); i++ {
			m := typ.Method(i)
			t.Errorf("\t%d: %s\n", i, m.Name)
		}
	}
	if got := typ.Method(0).Func.Call(args)[0]; got.Int() != 2 {
		t.Errorf("Wrong method table for outer, expected return value 2, got: %v", got)
	}
	if got := ValueOf((*outer).M).Call(args)[0]; got.Int() != 2 {
		t.Errorf("Wrong method table for outer, expected return value 2, got: %v", got)
	}
}

func TestEmbeddedMethods(t *testing.T) {
	// This test is similar to the upstream, but avoids using the unsupported
	// Value.UnsafePointer() method.
	typ := TypeOf((*OuterInt)(nil))
	if typ.NumMethod() != 1 {
		t.Errorf("Wrong method table for OuterInt: (m=%p)", (*OuterInt).M)
		for i := 0; i < typ.NumMethod(); i++ {
			m := typ.Method(i)
			t.Errorf("\t%d: %s %p\n", i, m.Name, m.Func.UnsafePointer())
		}
	}

	i := &InnerInt{3}
	if v := ValueOf(i).Method(0).Call(nil)[0].Int(); v != 3 {
		t.Errorf("i.M() = %d, want 3", v)
	}

	o := &OuterInt{1, InnerInt{2}}
	if v := ValueOf(o).Method(0).Call(nil)[0].Int(); v != 2 {
		t.Errorf("i.M() = %d, want 2", v)
	}

	f := (*OuterInt).M
	if v := f(o); v != 2 {
		t.Errorf("f(o) = %d, want 2", v)
	}
}

func TestNotInHeapDeref(t *testing.T) {
	t.Skip("GopherJS doesn't support //go:notinheap")
}

func TestMethodCallValueCodePtr(t *testing.T) {
	t.Skip("methodValueCallCodePtr() is not applicable in GopherJS")
}

func TestIssue50208(t *testing.T) {
	t.Skip("This test required generics, which are not yet supported: https://github.com/gopherjs/gopherjs/issues/1013")
}
