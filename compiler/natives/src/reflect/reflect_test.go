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

// Go 1.13
func TestMakeFuncInvalidReturnAssignments(t *testing.T) {
	t.Skip("TestMakeFuncInvalidReturnAssignments")
}

// Go 1.14
// func TestStructOfDifferentPkgPath(t *testing.T) {
// 	t.Skip("TestStructOfDifferentPkgPath")
// }

func init() {
	// TODO: This is a failure in 1.11, try to determine the cause and fix.
	typeTests = append(typeTests[:31], typeTests[32:]...) // skip test case #31
}

func TestStructOfDirectIface(t *testing.T) {
	t.Skip("reflect.InterfaceData unsupport")
}

func TestStructOfWithInterface(t *testing.T) {
	const want = 42
	type Iface interface {
		Get() int
	}
	type IfaceSet interface {
		Set(int)
	}
	tests := []struct {
		name string
		typ  Type
		val  Value
		impl bool
	}{
		{
			name: "StructI",
			typ:  TypeOf(StructI(want)),
			val:  ValueOf(StructI(want)),
			impl: true,
		},
		{
			name: "StructI",
			typ:  PtrTo(TypeOf(StructI(want))),
			val: ValueOf(func() interface{} {
				v := StructI(want)
				return &v
			}()),
			impl: true,
		},
		{
			name: "StructIPtr",
			typ:  PtrTo(TypeOf(StructIPtr(want))),
			val: ValueOf(func() interface{} {
				v := StructIPtr(want)
				return &v
			}()),
			impl: true,
		},
		{
			name: "StructIPtr",
			typ:  TypeOf(StructIPtr(want)),
			val:  ValueOf(StructIPtr(want)),
			impl: false,
		},
		// {
		// 	typ:  TypeOf((*Iface)(nil)).Elem(), // FIXME(sbinet): fix method.ifn/tfn
		// 	val:  ValueOf(StructI(want)),
		// 	impl: true,
		// },
	}

	for i, table := range tests {
		for j := 0; j < 2; j++ {
			var fields []StructField
			if j == 1 {
				fields = append(fields, StructField{
					Name:    "Dummy",
					PkgPath: "",
					Type:    TypeOf(int(0)),
				})
			}
			fields = append(fields, StructField{
				Name:      table.name,
				Anonymous: true,
				PkgPath:   "",
				Type:      table.typ,
			})

			// We currently do not correctly implement methods
			// for embedded fields other than the first.
			// Therefore, for now, we expect those methods
			// to not exist.  See issues 15924 and 20824.
			// When those issues are fixed, this test of panic
			// should be removed.
			if j == 1 && table.impl {
				func() {
					defer func() {
						if err := recover(); err == nil {
							t.Errorf("test-%d-%d did not panic", i, j)
						}
					}()
					_ = StructOf(fields)
				}()
				continue
			}

			rt := StructOf(fields)
			rv := New(rt).Elem()
			rv.Field(j).Set(table.val)

			if _, ok := rv.Interface().(Iface); ok != table.impl {
				if table.impl {
					t.Errorf("test-%d-%d: type=%v fails to implement Iface.\n", i, j, table.typ)
				} else {
					t.Errorf("test-%d-%d: type=%v should NOT implement Iface\n", i, j, table.typ)
				}
				continue
			}

			if !table.impl {
				continue
			}

			v := rv.Interface().(Iface).Get()
			if v != want {
				t.Errorf("test-%d-%d: x.Get()=%v. want=%v\n", i, j, v, want)
			}

			fct := rv.MethodByName("Get")
			out := fct.Call(nil)
			if !DeepEqual(out[0].Interface(), want) {
				t.Errorf("test-%d-%d: x.Get()=%v. want=%v\n", i, j, out[0].Interface(), want)
			}
		}
	}
	// Test an embedded nil pointer with pointer methods.
	fields := []StructField{{
		Name:      "StructIPtr",
		Anonymous: true,
		Type:      PtrTo(TypeOf(StructIPtr(want))),
	}}
	rt := StructOf(fields)
	rv := New(rt).Elem()
	// This should panic since the pointer is nil.
	_shouldPanic(func() {
		rv.Interface().(IfaceSet).Set(want)
	})

	// Test an embedded nil pointer to a struct with pointer methods.

	fields = []StructField{{
		Name:      "SettableStruct",
		Anonymous: true,
		Type:      PtrTo(TypeOf(SettableStruct{})),
	}}
	rt = StructOf(fields)
	rv = New(rt).Elem()
	// This should panic since the pointer is nil.
	_shouldPanic(func() {
		rv.Interface().(IfaceSet).Set(want)
	})

	// The behavior is different if there is a second field,
	// since now an interface value holds a pointer to the struct
	// rather than just holding a copy of the struct.
	fields = []StructField{
		{
			Name:      "SettableStruct",
			Anonymous: true,
			Type:      PtrTo(TypeOf(SettableStruct{})),
		},
		{
			Name:      "EmptyStruct",
			Anonymous: true,
			Type:      StructOf(nil),
		},
	}
	// With the current implementation this is expected to panic.
	// Ideally it should work and we should be able to see a panic
	// if we call the Set method.
	_shouldPanic(func() {
		StructOf(fields)
	})

	return
	// Embed a field that can be stored directly in an interface,
	// with a second field.
	fields = []StructField{
		{
			Name:      "SettablePointer",
			Anonymous: true,
			Type:      TypeOf(SettablePointer{}),
		},
		{
			Name:      "EmptyStruct",
			Anonymous: true,
			Type:      StructOf(nil),
		},
	}
	// With the current implementation this is expected to panic.
	// Ideally it should work and we should be able to call the
	// Set and Get methods.
	_shouldPanic(func() {
		StructOf(fields)
	})
}

func _shouldPanic(f func()) {
	defer func() {
		if recover() == nil {
			panic("did not panic")
		}
	}()
	f()
}
