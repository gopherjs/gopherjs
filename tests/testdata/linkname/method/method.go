package method

import (
	"strings"
	_ "unsafe"
)

type Point struct {
	X int
	Y int
}

func (pt *Point) Set(x, y int) {
	pt.X, pt.Y = x, y
}

func (pt Point) Get() (int, int) {
	return pt.X, pt.Y
}

//go:linkname struct_Set github.com/gopherjs/gopherjs/tests/testdata/linkname/method.(*Point).Set
func struct_Set(pt *point, x int, y int)

//go:linkname struct_Get github.com/gopherjs/gopherjs/tests/testdata/linkname/method.Point.Get
func struct_Get(pt point) (int, int)

type point struct {
	X int
	Y int
}

func testStruct() {
	var pt point
	struct_Set(&pt, 1, 2)
	x, y := struct_Get(pt)
	if x != 1 || y != 2 {
		panic(pt)
	}
}

type List []string

func (t *List) Append(s ...string) {
	*t = append(*t, s...)
}

func (t List) Get() string {
	return strings.Join(t, ",")
}

type list []string

//go:linkname slice_Append github.com/gopherjs/gopherjs/tests/testdata/linkname/method.(*List).Append
func slice_Append(*list, ...string)

//go:linkname slice_Get github.com/gopherjs/gopherjs/tests/testdata/linkname/method.List.Get
func slice_Get(list) string

func testSlice() {
	var v list
	v = append(v, "one")
	slice_Append(&v, "two", "three")
	s := slice_Get(v)
	if s != "one,two,three" {
		panic(s)
	}
}

type Array [5]string

func (t *Array) Set(i int, s string) {
	(*t)[i] = s
}

func (t Array) Get() string {
	return strings.Join(t[:], ",")
}

type array [5]string

//go:linkname array_Set github.com/gopherjs/gopherjs/tests/testdata/linkname/method.(*Array).Set
func array_Set(*array, int, string)

//go:linkname array_Get github.com/gopherjs/gopherjs/tests/testdata/linkname/method.Array.Get
func array_Get(array) string

func testArray() {
	var a array
	a[0] = "one"
	array_Set(&a, 1, "two")
	array_Set(&a, 4, "five")
	r := array_Get(a)
	if r != "one,two,,,five" {
		panic(r)
	}
}

type Map map[int]string

func (m Map) Set(key int, value string) {
	m[key] = value
}

func (m *Map) SetPtr(key int, value string) {
	(*m)[key] = value
}

func (m Map) Get() string {
	var list []string
	for _, v := range m {
		list = append(list, v)
	}
	return strings.Join(list, ",")
}

type _map map[int]string

//go:linkname map_Set github.com/gopherjs/gopherjs/tests/testdata/linkname/method.Map.Set
func map_Set(_map, int, string)

//go:linkname map_SetPtr github.com/gopherjs/gopherjs/tests/testdata/linkname/method.(*Map).SetPtr
func map_SetPtr(*_map, int, string)

//go:linkname map_Get github.com/gopherjs/gopherjs/tests/testdata/linkname/method.Map.Get
func map_Get(_map) string

func testMap() {
	m := make(_map)
	map_Set(m, 1, "one")
	map_SetPtr(&m, 2, "two")
	r := map_Get(m)
	if r != "one,two" {
		panic(r)
	}
}

type Func func(int, int) int

func (f Func) Call(a, b int) int {
	return f(a, b)
}

func (f *Func) CallPtr(a, b int) int {
	return (*f)(a, b)
}

type _func func(int, int) int

//go:linkname func_Call github.com/gopherjs/gopherjs/tests/testdata/linkname/method.Func.Call
func func_Call(_func, int, int) int

//go:linkname func_CallPtr github.com/gopherjs/gopherjs/tests/testdata/linkname/method.(*Func).CallPtr
func func_CallPtr(*_func, int, int) int

func testFunc() {
	var fn _func = func(a, b int) int {
		return a + b
	}
	r := func_Call(fn, 100, 200)
	if r != 300 {
		panic(r)
	}
	r2 := func_CallPtr(&fn, 100, 200)
	if r2 != 300 {
		panic(r2)
	}
}

type Chan chan int

func (c Chan) Send(n int) {
	c <- n
}

func (c *Chan) SendPtr(n int) {
	*c <- n
}

func (c Chan) Recv() int {
	return <-c
}

type _chan chan int

//go:linkname chan_Send github.com/gopherjs/gopherjs/tests/testdata/linkname/method.Chan.Send
func chan_Send(_chan, int)

//go:linkname chan_SendPtr github.com/gopherjs/gopherjs/tests/testdata/linkname/method.(*Chan).SendPtr
func chan_SendPtr(*_chan, int)

//go:linkname chan_Recv github.com/gopherjs/gopherjs/tests/testdata/linkname/method.Chan.Recv
func chan_Recv(_chan) int

func testChan() {
	c := make(_chan)
	go func() {
		chan_Send(c, 100)
	}()
	r := chan_Recv(c)
	if r != 100 {
		panic(r)
	}
	go func() {
		chan_SendPtr(&c, 200)
	}()
	r = chan_Recv(c)
	if r != 200 {
		panic(r)
	}
}

type T = complex64

type Basic int

func (m *Basic) Set(v int) {
	*m = Basic(v)
}

func (m Basic) Get() int {
	return int(m)
}

type basic uintptr

//go:linkname basic_Set github.com/gopherjs/gopherjs/tests/testdata/linkname/method.(*Basic).Set
func basic_Set(*_int, int) int

//go:linkname basic_Get github.com/gopherjs/gopherjs/tests/testdata/linkname/method.Basic.Get
func basic_Get(_int) int

type Int int

func (m *Int) Set(v int) {
	*m = Int(v)
}

func (m Int) Get() int {
	return int(m)
}

type _int int

//go:linkname int_Set github.com/gopherjs/gopherjs/tests/testdata/linkname/method.(*Int).Set
func int_Set(*_int, int) int

//go:linkname int_Get github.com/gopherjs/gopherjs/tests/testdata/linkname/method.Int.Get
func int_Get(_int) int

func testInt() {
	var i _int
	int_Set(&i, 100)
	r := int_Get(i)
	if r != 100 {
		panic(r)
	}
}

type Uint uint

func (m *Uint) Set(v uint) {
	*m = Uint(v)
}

func (m Uint) Get() uint {
	return uint(m)
}

type _uint uint

//go:linkname uint_Set github.com/gopherjs/gopherjs/tests/testdata/linkname/method.(*Uint).Set
func uint_Set(*_uint, uint) uint

//go:linkname uint_Get github.com/gopherjs/gopherjs/tests/testdata/linkname/method.Uint.Get
func uint_Get(_uint) uint

func testUint() {
	var i _uint
	uint_Set(&i, 100)
	r := uint_Get(i)
	if r != 100 {
		panic(r)
	}
}

type Float64 float64

func (m *Float64) Set(v float64) {
	*m = Float64(v)
}

func (m Float64) Get() float64 {
	return float64(m)
}

type _float64 float64

//go:linkname float64_Set github.com/gopherjs/gopherjs/tests/testdata/linkname/method.(*Float64).Set
func float64_Set(*_float64, float64) float64

//go:linkname float64_Get github.com/gopherjs/gopherjs/tests/testdata/linkname/method.Float64.Get
func float64_Get(_float64) float64

func testFloat64() {
	var i _float64
	float64_Set(&i, 3.14)
	r := float64_Get(i)
	if r != 3.14 {
		panic(r)
	}
}

type Complex128 complex128

func (m *Complex128) Set(v complex128) {
	*m = Complex128(v)
}

func (m Complex128) Get() complex128 {
	return complex128(m)
}

type _complex128 complex128

//go:linkname complex128_Set github.com/gopherjs/gopherjs/tests/testdata/linkname/method.(*Complex128).Set
func complex128_Set(*_complex128, complex128) complex128

//go:linkname complex128_Get github.com/gopherjs/gopherjs/tests/testdata/linkname/method.Complex128.Get
func complex128_Get(_complex128) complex128

func testComplex128() {
	var i _complex128
	complex128_Set(&i, 1+2i)
	r := complex128_Get(i)
	if r != 1+2i {
		panic(r)
	}
}

type Uintptr uintptr

func (m *Uintptr) Set(v uintptr) {
	*m = Uintptr(v)
}

func (m Uintptr) Get() uintptr {
	return uintptr(m)
}

type _uintptr uintptr

//go:linkname uintptr_Set github.com/gopherjs/gopherjs/tests/testdata/linkname/method.(*Uintptr).Set
func uintptr_Set(*_uintptr, uintptr) uintptr

//go:linkname uintptr_Get github.com/gopherjs/gopherjs/tests/testdata/linkname/method.Uintptr.Get
func uintptr_Get(_uintptr) uintptr

func testUintptr() {
	var i _uintptr
	uintptr_Set(&i, 0x1234)
	r := uintptr_Get(i)
	if r != 0x1234 {
		panic(r)
	}
}

type Bool bool

func (m *Bool) Set(v bool) {
	*m = Bool(v)
}

func (m Bool) Get() bool {
	return bool(m)
}

type _bool bool

//go:linkname bool_Set github.com/gopherjs/gopherjs/tests/testdata/linkname/method.(*Bool).Set
func bool_Set(*_bool, bool) bool

//go:linkname bool_Get github.com/gopherjs/gopherjs/tests/testdata/linkname/method.Bool.Get
func bool_Get(_bool) bool

func testBool() {
	var i _bool
	bool_Set(&i, true)
	r := bool_Get(i)
	if r != true {
		panic(r)
	}
}

type Byte byte

func (m *Byte) Set(v byte) {
	*m = Byte(v)
}

func (m Byte) Get() byte {
	return byte(m)
}

type _byte byte

//go:linkname byte_Set github.com/gopherjs/gopherjs/tests/testdata/linkname/method.(*Byte).Set
func byte_Set(*_byte, byte) byte

//go:linkname byte_Get github.com/gopherjs/gopherjs/tests/testdata/linkname/method.Byte.Get
func byte_Get(_byte) byte

func testByte() {
	var i _byte
	byte_Set(&i, 0x7f)
	r := byte_Get(i)
	if r != 0x7f {
		panic(r)
	}
}

type String string

func (m *String) Set(v string) {
	*m = String(v)
}

func (m String) Get() string {
	return string(m)
}

type _string string

//go:linkname string_Set github.com/gopherjs/gopherjs/tests/testdata/linkname/method.(*String).Set
func string_Set(*_string, string) string

//go:linkname string_Get github.com/gopherjs/gopherjs/tests/testdata/linkname/method.String.Get
func string_Get(_string) string

func testString() {
	var i _string
	string_Set(&i, "hello world")
	r := string_Get(i)
	if r != "hello world" {
		panic(r)
	}
}

func TestLinkname() {
	testStruct()
	testSlice()
	testArray()
	testMap()
	testFunc()
	testChan()
	testBool()
	testByte()
	testInt()
	testUint()
	testFloat64()
	testComplex128()
	testString()
}
