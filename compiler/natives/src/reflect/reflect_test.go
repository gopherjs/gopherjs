//go:build js
// +build js

package reflect_test

import (
	"math"
	"reflect"
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
	reflect.Select([]reflect.SelectCase{
		{
			Dir:  reflect.SelectRecv,
			Chan: reflect.Value{},
		}, {
			Dir:  reflect.SelectSend,
			Chan: reflect.Value{},
			Send: reflect.ValueOf(1),
		}, {
			Dir: reflect.SelectDefault,
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
	m := reflect.ValueOf(NonExportedFirst(0)).Method(0)

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
	y := x.Convert(reflect.TypeOf(float32(0)))
	z := y.Interface().(float32)
	if got := math.Float32bits(z); got != qnan {
		t.Errorf("quiet nan conversion got %x, want %x", got, qnan)
	}
}
