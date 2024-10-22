package analysis

import (
	"go/ast"
	"go/types"
	"testing"

	"github.com/gopherjs/gopherjs/compiler/internal/typeparams"
	"github.com/gopherjs/gopherjs/internal/srctesting"
)

func TestBlocking_Simple(t *testing.T) {
	bt := newBlockingTest(t,
		`package test

		func notBlocking() {
			println("hi")
		}`)
	bt.assertNotBlocking(`notBlocking`)
}

func TestBlocking_Recursive(t *testing.T) {
	bt := newBlockingTest(t,
		`package test

		func notBlocking(i int) {
			if i > 0 {
				println(i)
				notBlocking(i - 1)
			}
		}`)
	bt.assertNotBlocking(`notBlocking`)
}

func TestBlocking_AlternatingRecursive(t *testing.T) {
	bt := newBlockingTest(t,
		`package test

		func near(i int) {
			if i > 0 {
				println(i)
				far(i)
			}
		}

		func far(i int) {
			near(i - 1)
		}`)
	bt.assertNotBlocking(`near`)
	bt.assertNotBlocking(`far`)
}

func TestBlocking_Channels(t *testing.T) {
	bt := newBlockingTest(t,
		`package test

		func readFromChannel(c chan bool) {
			<-c
		}

		func readFromChannelAssign(c chan bool) {
			v := <-c
			println(v)
		}

		func readFromChannelAsArg(c chan bool) {
			println(<-c)
		}

		func sendToChannel(c chan bool) {
			c <- true
		}

		func rangeOnChannel(c chan bool) {
			for v := range c {
				println(v)
			}
		}

		func rangeOnSlice(c []bool) {
			for v := range c {
				println(v)
			}
		}`)
	bt.assertBlocking(`readFromChannel`)
	bt.assertBlocking(`sendToChannel`)
	bt.assertBlocking(`rangeOnChannel`)
	bt.assertBlocking(`readFromChannelAssign`)
	bt.assertBlocking(`readFromChannelAsArg`)
	bt.assertNotBlocking(`rangeOnSlice`)
}

func TestBlocking_Selects(t *testing.T) {
	bt := newBlockingTest(t,
		`package test

		func selectReadWithoutDefault(a, b chan bool) {
			select {
			case <-a:
				println("a")
			case v := <-b:
				println("b", v)
			}
		}

		func selectReadWithDefault(a, b chan bool) {
			select {
			case <-a:
				println("a")
			case v := <-b:
				println("b", v)
			default:
				println("nothing")
			}
		}

		func selectSendWithoutDefault(a, b chan bool) {
			select {
			case a <- true:
				println("a")
			case b <- false:
				println("b")
			}
		}

		func selectSendWithDefault(a, b chan bool) {
			select {
			case a <- true:
				println("a")
			case b <- false:
				println("b")
			default:
				println("nothing")
			}
		}`)
	bt.assertBlocking(`selectReadWithoutDefault`)
	bt.assertBlocking(`selectSendWithoutDefault`)
	bt.assertNotBlocking(`selectReadWithDefault`)
	bt.assertNotBlocking(`selectSendWithDefault`)
}

func TestBlocking_GoRoutines_WithFuncLiterals(t *testing.T) {
	bt := newBlockingTest(t,
		`package test

		func notBlocking(c chan bool) {
			go func(c chan bool) { // line 4
				println(<-c)
			}(c)
		}

		func blocking(c chan bool) {
			go func(v bool) { // line 10
				println(v)
			}(<-c)
		}`)
	bt.assertNotBlocking(`notBlocking`)
	bt.assertBlockingLit(4)

	bt.assertBlocking(`blocking`)
	bt.assertNotBlockingLit(10)
}

func TestBlocking_GoRoutines_WithNamedFuncs(t *testing.T) {
	bt := newBlockingTest(t,
		`package test

		func blockingRoutine(c chan bool) {
			println(<-c)
		}

		func nonBlockingRoutine(v bool) {
			println(v)
		}

		func notBlocking(c chan bool) {
			go blockingRoutine(c)
		}

		func blocking(c chan bool) {
			go nonBlockingRoutine(<-c)
		}`)
	bt.assertBlocking(`blockingRoutine`)
	bt.assertNotBlocking(`nonBlockingRoutine`)

	bt.assertNotBlocking(`notBlocking`)
	bt.assertBlocking(`blocking`)
}

func TestBlocking_Defers_WithoutReturns_WithFuncLiterals(t *testing.T) {
	bt := newBlockingTest(t,
		`package test

		func blockingBody(c chan bool) {
			defer func(c chan bool) { // line 4
				println(<-c)
			}(c)
		}

		func blockingArg(c chan bool) {
			defer func(v bool) { // line 10
				println(v)
			}(<-c)
		}

		func notBlocking(c chan bool) {
			defer func(v bool) { // line 16
				println(v)
			}(true)
		}`)
	bt.assertBlocking(`blockingBody`)
	bt.assertBlockingLit(4)

	bt.assertBlocking(`blockingArg`)
	bt.assertNotBlockingLit(10)

	bt.assertNotBlocking(`notBlocking`)
	bt.assertNotBlockingLit(16)
}

func TestBlocking_Defers_WithoutReturns_WithNamedFuncs(t *testing.T) {
	bt := newBlockingTest(t,
		`package test

		func blockingPrint(c chan bool) {
			println(<-c)
		}

		func nonBlockingPrint(v bool) {
			println(v)
		}

		func blockingBody(c chan bool) {
			defer blockingPrint(c)
		}

		func blockingArg(c chan bool) {
			defer nonBlockingPrint(<-c)
		}

		func notBlocking(c chan bool) {
			defer nonBlockingPrint(true)
		}`)
	bt.assertBlocking(`blockingPrint`)
	bt.assertNotBlocking(`nonBlockingPrint`)

	bt.assertBlocking(`blockingBody`)
	bt.assertBlocking(`blockingArg`)
	bt.assertNotBlocking(`notBlocking`)
}

func TestBlocking_Defers_WithReturns_WithFuncLiterals(t *testing.T) {
	bt := newBlockingTest(t,
		`package test

		func blockingBody(c chan bool) int {
			defer func(c chan bool) { // line 4
				println(<-c)
			}(c)
			return 42
		}

		func blockingArg(c chan bool) int {
			defer func(v bool) { // line 11
				println(v)
			}(<-c)
			return 42
		}

		func notBlocking(c chan bool) int {
			defer func(v bool) { // line 18
				println(v)
			}(true)
			return 42
		}`)
	bt.assertBlocking(`blockingBody`)
	bt.assertBlockingLit(4)

	bt.assertBlocking(`blockingArg`)
	bt.assertNotBlockingLit(11)

	// TODO: The following is blocking because currently any defer with a return
	// is assumed to be blocking. This limitation should be fixed in the future.
	bt.assertBlocking(`notBlocking`)
	bt.assertNotBlockingLit(18)
}

func TestBlocking_Defers_WithReturns_WithNamedFuncs(t *testing.T) {
	bt := newBlockingTest(t,
		`package test

		func blockingPrint(c chan bool) {
			println(<-c)
		}

		func nonBlockingPrint(v bool) {
			println(v)
		}

		func blockingBody(c chan bool) int {
			defer blockingPrint(c)
			return 42
		}

		func blockingArg(c chan bool) int {
			defer nonBlockingPrint(<-c)
			return 42
		}

		func notBlocking(c chan bool) int {
			defer nonBlockingPrint(true)
			return 42
		}`)
	bt.assertBlocking(`blockingPrint`)
	bt.assertNotBlocking(`nonBlockingPrint`)

	bt.assertBlocking(`blockingBody`)
	bt.assertBlocking(`blockingArg`)

	// TODO: The following is blocking because currently any defer with a return
	// is assumed to be blocking. This limitation should be fixed in the future.
	bt.assertBlocking(`notBlocking`)
}

func TestBlocking_Returns_WithoutDefers(t *testing.T) {
	bt := newBlockingTest(t,
		`package test

		func blocking(c chan bool) bool {
			return <-c
		}

		func indirectlyBlocking(c chan bool) bool {
			return blocking(c)
		}

		func notBlocking(c chan bool) bool {
			return true
		}`)
	bt.assertBlocking(`blocking`)
	bt.assertBlocking(`indirectlyBlocking`)
	bt.assertNotBlocking(`notBlocking`)
}

func TestBlocking_FunctionLiteral(t *testing.T) {
	// See: https://github.com/gopherjs/gopherjs/issues/955.
	bt := newBlockingTest(t,
		`package test

		func blocking() {
			c := make(chan bool)
			<-c
		}

		func indirectlyBlocking() {
			func() { blocking() }() // line 9
		}

		func directlyBlocking() {
			func() {  // line 13
				c := make(chan bool)
				<-c
			}()
		}

		func notBlocking() {
			func() { println() } () // line 20
		}`)
	bt.assertBlocking(`blocking`)

	bt.assertBlocking(`indirectlyBlocking`)
	bt.assertBlockingLit(9)

	bt.assertBlocking(`directlyBlocking`)
	bt.assertBlockingLit(13)

	bt.assertNotBlocking(`notBlocking`)
	bt.assertNotBlockingLit(20)
}

func TestBlocking_LinkedFunction(t *testing.T) {
	bt := newBlockingTest(t,
		`package test

		// linked to some other function
		func blocking()

		func indirectlyBlocking() {
			blocking()
		}`)
	bt.assertBlocking(`blocking`)
	bt.assertBlocking(`indirectlyBlocking`)
}

func TestBlocking_Instances_WithSingleTypeArg(t *testing.T) {
	bt := newBlockingTest(t,
		`package test

		func blocking[T any]() {
			c := make(chan T)
			<-c
		}

		func notBlocking[T any]() {
			var v T
			println(v)
		}

		func bInt() {
			blocking[int]()
		}

		func nbUint() {
			notBlocking[uint]()
		}`)
	bt.assertFuncInstCount(4)
	// blocking and notBlocking as generics do not have FuncInfo,
	// only non-generic and instances have FuncInfo.

	bt.assertBlockingInst(`test.blocking[int]`)
	bt.assertBlocking(`bInt`)
	bt.assertNotBlockingInst(`test.notBlocking[uint]`)
	bt.assertNotBlocking(`nbUint`)
}

func TestBlocking_Instances_WithMultipleTypeArgs(t *testing.T) {
	bt := newBlockingTest(t,
		`package test

		func blocking[K comparable, V any, M ~map[K]V]() {
			c := make(chan M)
			<-c
		}

		func notBlocking[K comparable, V any, M ~map[K]V]() {
			var m M
			println(m)
		}

		func bInt() {
			blocking[string, int, map[string]int]()
		}

		func nbUint() {
			notBlocking[string, uint, map[string]uint]()
		}`)
	bt.assertFuncInstCount(4)
	// blocking and notBlocking as generics do not have FuncInfo,
	// only non-generic and instances have FuncInfo.

	bt.assertBlockingInst(`test.blocking[string, int, map[string]int]`)
	bt.assertBlocking(`bInt`)
	bt.assertNotBlockingInst(`test.notBlocking[string, uint, map[string]uint]`)
	bt.assertNotBlocking(`nbUint`)
}

func TestBlocking_Indexed_FunctionSlice(t *testing.T) {
	// This calls notBlocking but since the function pointers
	// are in the slice they will both be considered as blocking.
	bt := newBlockingTest(t,
		`package test

		func blocking() {
			c := make(chan int)
			<-c
		}

		func notBlocking() {
			println()
		}

		var funcs = []func() { blocking, notBlocking }

		func indexer(i int) {
			funcs[i]()
		}`)
	bt.assertBlocking(`blocking`)
	bt.assertBlocking(`indexer`)
	bt.assertNotBlocking(`notBlocking`)
}

func TestBlocking_Indexed_FunctionMap(t *testing.T) {
	// This calls notBlocking but since the function pointers
	// are in the map they will both be considered as blocking.
	bt := newBlockingTest(t,
		`package test

		func blocking() {
			c := make(chan int)
			<-c
		}

		func notBlocking() {
			println()
		}

		var funcs = map[string]func() {
			"b":  blocking,
			"nb": notBlocking,
		}

		func indexer(key string) {
			funcs[key]()
		}`)
	bt.assertBlocking(`blocking`)
	bt.assertBlocking(`indexer`)
	bt.assertNotBlocking(`notBlocking`)
}

func TestBlocking_Indexed_FunctionArray(t *testing.T) {
	// This calls notBlocking but since the function pointers
	// are in the array they will both be considered as blocking.
	bt := newBlockingTest(t,
		`package test

		func blocking() {
			c := make(chan int)
			<-c
		}

		func notBlocking() {
			println()
		}

		var funcs = [2]func() { blocking, notBlocking }

		func indexer(i int) {
			funcs[i]()
		}`)
	bt.assertBlocking(`blocking`)
	bt.assertBlocking(`indexer`)
	bt.assertNotBlocking(`notBlocking`)
}

func TestBlocking_Casting_InterfaceInstanceWithSingleTypeParam(t *testing.T) {
	// This checks that casting to an instance type with a single type parameter
	// is treated as a cast and not accidentally treated as a function call.
	bt := newBlockingTest(t,
		`package test

		type Foo[T any] interface {
			Baz() T
		}

		type Bar struct {
			name string
		}

		func (b Bar) Baz() string {
			return b.name
		}

		func caster() Foo[string] {
			b := Bar{name: "foo"}
			return Foo[string](b)
		}`)
	bt.assertNotBlocking(`caster`)
}

func TestBlocking_Casting_InterfaceInstanceWithMultipleTypeParams(t *testing.T) {
	// This checks that casting to an instance type with multiple type parameters
	// is treated as a cast and not accidentally treated as a function call.
	bt := newBlockingTest(t,
		`package test

		type Foo[K comparable, V any] interface {
			Baz(K) V
		}

		type Bar struct {
			dat map[string]int
		}

		func (b Bar) Baz(key string) int {
			return b.dat[key]
		}

		func caster() Foo[string, int] {
			b := Bar{ dat: map[string]int{ "foo": 2 }}
			return Foo[string, int](b)
		}`)
	bt.assertNotBlocking(`caster`)
}

func TestBlocking_Casting_Interface(t *testing.T) {
	// This checks that non-generic casting of type is treated as a
	// cast and not accidentally treated as a function call.
	bt := newBlockingTest(t,
		`package test
	
		type Foo interface {
			Baz() string
		}

		type Bar struct {
			name string
		}

		func (b Bar) Baz() string {
			return b.name
		}

		func caster() Foo {
			b := Bar{"foo"}
			return Foo(b)
		}`)
	bt.assertNotBlocking(`caster`)
}

func TestBlocking_ComplexCasting(t *testing.T) {
	// This checks a complex casting to a type is treated as a
	// cast and not accidentally treated as a function call.
	bt := newBlockingTest(t,
		`package test
	
		type Foo interface {
			Bar() string
		}
		
		func doNothing(f Foo) Foo  {
			return interface{ Bar() string }(f)
		}`)
	bt.assertNotBlocking(`doNothing`)
}

func TestBlocking_ComplexCall(t *testing.T) {
	// This checks a complex call of a function is defaulted to blocking.
	bt := newBlockingTest(t,
		`package test
	
		type Foo func() string
		
		func bar(f any) string  {
			return f.(Foo)()
		}`)
	bt.assertBlocking(`bar`)
}

func TestBlocking_CallWithNamedInterfaceReceiver(t *testing.T) {
	// This checks that calling a named interface function is defaulted to blocking.
	bt := newBlockingTest(t,
		`package test
	
		type Foo interface {
			Baz()
		}
		
		func bar(f Foo) {
			f.Baz()
		}`)
	bt.assertBlocking(`bar`)
}

func TestBlocking_CallWithUnnamedInterfaceReceiver(t *testing.T) {
	// This checks that calling an unnamed interface function is defaulted to blocking.
	bt := newBlockingTest(t,
		`package test
			
		func bar(f interface { Baz() }) {
			f.Baz()
		}`)
	bt.assertBlocking(`bar`)
}

func TestBlocking_VarFunctionCall(t *testing.T) {
	// This checks that calling a function in a var is defaulted to blocking.
	bt := newBlockingTest(t,
		`package test
	
		var foo = func() { // line 3
			println("hi")
		}
		
		func bar() {
			foo()
		}`)
	bt.assertNotBlockingLit(3)
	bt.assertBlocking(`bar`)
}

func TestBlocking_FieldFunctionCallOnNamed(t *testing.T) {
	// This checks that calling a function in a field is defaulted to blocking.
	// This should be the same as the previous test but with a field since
	// all function pointers are treated as blocking.
	bt := newBlockingTest(t,
		`package test
	
		type foo struct {
			Baz func()
		}
		
		func bar(f foo) {
			f.Baz()
		}`)
	bt.assertBlocking(`bar`)
}

func TestBlocking_FieldFunctionCallOnUnnamed(t *testing.T) {
	// Same as previous test but with an unnamed struct.
	bt := newBlockingTest(t,
		`package test
	
		func bar(f struct { Baz func() }) {
			f.Baz()
		}`)
	bt.assertBlocking(`bar`)
}

func TestBlocking_ParamFunctionCall(t *testing.T) {
	// Same as previous test but with an unnamed function parameter.
	bt := newBlockingTest(t,
		`package test
	
		func bar(baz func()) {
			baz()
		}`)
	bt.assertBlocking(`bar`)
}

func TestBlocking_FunctionUnwrapping(t *testing.T) {
	// Test that calling a function that calls a function etc.
	// is defaulted to blocking.
	bt := newBlockingTest(t,
		`package test
	
		func bar(baz func()func()func()) {
			baz()()()
		}`)
	bt.assertBlocking(`bar`)
}

func TestBlocking_MethodCall_NonPointer(t *testing.T) {
	// Test that calling a method on a non-pointer receiver.
	bt := newBlockingTest(t,
		`package test
	
		type Foo struct {}

		func (f Foo) blocking() {
			ch := make(chan bool)
			<-ch
		}

		func (f Foo) notBlocking() {
			println("hi")
		}

		func blocking(f Foo) {
			f.blocking()
		}
			
		func notBlocking(f Foo) {
			f.notBlocking()
		}`)
	bt.assertBlocking(`Foo.blocking`)
	bt.assertNotBlocking(`Foo.notBlocking`)
	bt.assertBlocking(`blocking`)
	bt.assertNotBlocking(`notBlocking`)
}

func TestBlocking_MethodCall_Pointer(t *testing.T) {
	// Test that calling a method on a pointer receiver.
	bt := newBlockingTest(t,
		`package test
	
		type Foo struct {}

		func (f *Foo) blocking() {
			ch := make(chan bool)
			<-ch
		}

		func (f *Foo) notBlocking() {
			println("hi")
		}

		func blocking(f *Foo) {
			f.blocking()
		}
			
		func notBlocking(f *Foo) {
			f.notBlocking()
		}`)
	bt.assertBlocking(`Foo.blocking`)
	bt.assertNotBlocking(`Foo.notBlocking`)
	bt.assertBlocking(`blocking`)
	bt.assertNotBlocking(`notBlocking`)
}

func TestBlocking_InstantiationBlocking(t *testing.T) {
	// This checks that the instantiation of a generic function is
	// being used when checking for blocking not the type argument interface.
	bt := newBlockingTest(t,
		`package test
		
		type BazBlocker struct {
			c chan bool
		}
		func (bb BazBlocker) Baz() {
			println(<-bb.c)
		}

		type BazNotBlocker struct {}
		func (bnb BazNotBlocker) Baz() {
			println("hi")
		}

		type Foo interface { Baz() }
		func FooBaz[T Foo](foo T) {
			foo.Baz()
		}

		func blockingViaExplicit() {
			FooBaz[BazBlocker](BazBlocker{c: make(chan bool)})
		}
		
		func notBlockingViaExplicit() {
			FooBaz[BazNotBlocker](BazNotBlocker{})
		}

		func blockingViaImplicit() {
			FooBaz(BazBlocker{c: make(chan bool)})
		}
		
		func notBlockingViaImplicit() {
			FooBaz(BazNotBlocker{})
		}`)
	bt.assertFuncInstCount(8)
	// `FooBaz` as a generic function does not have FuncInfo for it,
	// only non-generic or instantiations of a generic functions have FuncInfo.

	bt.assertBlocking(`BazBlocker.Baz`)
	bt.assertBlocking(`blockingViaExplicit`)
	bt.assertBlocking(`blockingViaImplicit`)
	bt.assertBlockingInst(`test.FooBaz[pkg/test.BazBlocker]`)

	bt.assertNotBlocking(`BazNotBlocker.Baz`)
	bt.assertNotBlocking(`notBlockingViaExplicit`)
	bt.assertNotBlocking(`notBlockingViaImplicit`)
	bt.assertNotBlockingInst(`test.FooBaz[pkg/test.BazNotBlocker]`)
}

func TestBlocking_NestedInstantiations(t *testing.T) {
	// Checking that the type parameters are being propagated down into calls.
	bt := newBlockingTest(t,
		`package test
		
		func Foo[T any](t T) {
			println(t)
		}

		func Bar[K comparable, V any, M ~map[K]V](m M) {
			Foo(m)
		}

		func Baz[T any, S ~[]T](s S) {
			m:= map[int]T{}
			for i, v := range s {
				m[i] = v
			}
			Bar(m)
		}

		func bazInt() {
			Baz([]int{1, 2, 3})
		}
		
		func bazString() {
			Baz([]string{"one", "two", "three"})
		}`)
	bt.assertFuncInstCount(8)
	bt.assertNotBlocking(`bazInt`)
	bt.assertNotBlocking(`bazString`)
	bt.assertNotBlockingInst(`test.Foo[map[int]int]`)
	bt.assertNotBlockingInst(`test.Foo[map[int]string]`)
	bt.assertNotBlockingInst(`test.Bar[int, int, map[int]int]`)
	bt.assertNotBlockingInst(`test.Bar[int, string, map[int]string]`)
	bt.assertNotBlockingInst(`test.Baz[int, []int]`)
	bt.assertNotBlockingInst(`test.Baz[string, []string]`)
}

func TestBlocking_MethodSelection(t *testing.T) {
	// This tests method selection using method expression (receiver as the first
	// argument) selecting on type and method call selecting on a variable.
	// This tests in both generic (FooBaz[T]) and non-generic contexts.
	bt := newBlockingTest(t,
		`package test

		type Foo interface { Baz() }

		type BazBlocker struct {
			c chan bool
		}
		func (bb BazBlocker) Baz() {
			println(<-bb.c)
		}

		type BazNotBlocker struct {}
		func (bnb BazNotBlocker) Baz() {
			println("hi")
		}

		type FooBaz[T Foo] struct {}
		func (fb FooBaz[T]) ByMethodExpression() {
			var foo T
			T.Baz(foo)
		}
		func (fb FooBaz[T]) ByInstance() {
			var foo T
			foo.Baz()
		}

		func blocking() {
			fb := FooBaz[BazBlocker]{}

			FooBaz[BazBlocker].ByMethodExpression(fb)
			FooBaz[BazBlocker].ByInstance(fb)

			fb.ByMethodExpression()
			fb.ByInstance()
		}

		func notBlocking() {
			fb := FooBaz[BazNotBlocker]{}

			FooBaz[BazNotBlocker].ByMethodExpression(fb)
			FooBaz[BazNotBlocker].ByInstance(fb)

			fb.ByMethodExpression()
			fb.ByInstance()
		}`)
	bt.assertFuncInstCount(8)

	bt.assertBlocking(`BazBlocker.Baz`)
	bt.assertBlockingInst(`test.ByMethodExpression[pkg/test.BazBlocker]`)
	bt.assertBlockingInst(`test.ByInstance[pkg/test.BazBlocker]`)
	bt.assertBlocking(`blocking`)

	bt.assertNotBlocking(`BazNotBlocker.Baz`)
	bt.assertNotBlockingInst(`test.ByMethodExpression[pkg/test.BazNotBlocker]`)
	bt.assertNotBlockingInst(`test.ByInstance[pkg/test.BazNotBlocker]`)
	bt.assertNotBlocking(`notBlocking`)
}

func TestBlocking_IsImportBlocking_Simple(t *testing.T) {
	otherSrc := `package other

		func Blocking() {
			ch := make(chan bool)
			<-ch
		}

		func NotBlocking() {
			println("hi")
		}`

	testSrc := `package test

		import "pkg/other"

		func blocking() {
			other.Blocking()
		}
			
		func notBlocking() {
			other.NotBlocking()
		}`

	bt := newBlockingTestWithOtherPackage(t, testSrc, otherSrc)
	bt.assertBlocking(`blocking`)
	bt.assertNotBlocking(`notBlocking`)
}

func TestBlocking_IsImportBlocking_ForwardInstances(t *testing.T) {
	otherSrc := `package other

		type BazBlocker struct {
			c chan bool
		}
		func (bb BazBlocker) Baz() {
			println(<-bb.c)
		}

		type BazNotBlocker struct {}
		func (bnb BazNotBlocker) Baz() {
			println("hi")
		}`

	testSrc := `package test

		import "pkg/other"

		type Foo interface { Baz() }
		func FooBaz[T Foo](f T) {
			f.Baz()
		}

		func blocking() {
			FooBaz(other.BazBlocker{})
		}

		func notBlocking() {
			FooBaz(other.BazNotBlocker{})
		}`

	bt := newBlockingTestWithOtherPackage(t, testSrc, otherSrc)
	bt.assertBlocking(`blocking`)
	bt.assertNotBlocking(`notBlocking`)
}

func TestBlocking_IsImportBlocking_BackwardInstances(t *testing.T) {
	t.Skip(`isImportedBlocking doesn't fully handle instances yet`)
	// TODO(grantnelson-wf): This test is currently failing because the info
	// for the test package is need while creating the instances for FooBaz
	// while analyzing the other package. However the other package is analyzed
	// first since the test package is dependent on it. One possible fix is that
	// we add some mechanism similar to the localInstCallees but for remote
	// instances then perform the blocking propagation steps for all packages
	// including the localInstCallees propagation at the same time. After all the
	// propagation of the calls then the flow control statements can be marked.

	otherSrc := `package other

		type Foo interface { Baz() }
		func FooBaz[T Foo](f T) {
			f.Baz()
		}`

	testSrc := `package test

		import "pkg/other"

		type BazBlocker struct {
			c chan bool
		}
		func (bb BazBlocker) Baz() {
			println(<-bb.c)
		}

		type BazNotBlocker struct {}
		func (bnb BazNotBlocker) Baz() {
			println("hi")
		}

		func blocking() {
			other.FooBaz(BazBlocker{})
		}

		func notBlocking() {
			other.FooBaz(BazNotBlocker{})
		}`

	bt := newBlockingTestWithOtherPackage(t, testSrc, otherSrc)
	bt.assertBlocking(`blocking`)
	bt.assertNotBlocking(`notBlocking`)
}

type blockingTest struct {
	f       *srctesting.Fixture
	file    *ast.File
	pkgInfo *Info
}

func newBlockingTest(t *testing.T, src string) *blockingTest {
	f := srctesting.New(t)
	tc := typeparams.Collector{
		TContext:  types.NewContext(),
		Info:      f.Info,
		Instances: &typeparams.PackageInstanceSets{},
	}

	file := f.Parse(`test.go`, src)
	testInfo, testPkg := f.Check(`pkg/test`, file)
	tc.Scan(testPkg, file)

	isImportBlocking := func(i typeparams.Instance) bool {
		t.Fatalf(`isImportBlocking should not be called in this test, called with %v`, i)
		return true
	}
	pkgInfo := AnalyzePkg([]*ast.File{file}, f.FileSet, testInfo, types.NewContext(), testPkg, tc.Instances, isImportBlocking)

	return &blockingTest{
		f:       f,
		file:    file,
		pkgInfo: pkgInfo,
	}
}

func newBlockingTestWithOtherPackage(t *testing.T, testSrc string, otherSrc string) *blockingTest {
	f := srctesting.New(t)
	tc := typeparams.Collector{
		TContext:  types.NewContext(),
		Info:      f.Info,
		Instances: &typeparams.PackageInstanceSets{},
	}

	pkgInfo := map[*types.Package]*Info{}
	isImportBlocking := func(i typeparams.Instance) bool {
		if info, ok := pkgInfo[i.Object.Pkg()]; ok {
			return info.IsBlocking(i)
		}
		t.Fatalf(`unexpected package in isImportBlocking for %v`, i)
		return true
	}

	otherFile := f.Parse(`other.go`, otherSrc)
	_, otherPkg := f.Check(`pkg/other`, otherFile)
	tc.Scan(otherPkg, otherFile)

	testFile := f.Parse(`test.go`, testSrc)
	_, testPkg := f.Check(`pkg/test`, testFile)
	tc.Scan(testPkg, testFile)

	otherPkgInfo := AnalyzePkg([]*ast.File{otherFile}, f.FileSet, f.Info, types.NewContext(), otherPkg, tc.Instances, isImportBlocking)
	pkgInfo[otherPkg] = otherPkgInfo

	testPkgInfo := AnalyzePkg([]*ast.File{testFile}, f.FileSet, f.Info, types.NewContext(), testPkg, tc.Instances, isImportBlocking)
	pkgInfo[testPkg] = testPkgInfo

	return &blockingTest{
		f:       f,
		file:    testFile,
		pkgInfo: testPkgInfo,
	}
}

func (bt *blockingTest) assertFuncInstCount(expCount int) {
	if got := bt.pkgInfo.funcInstInfos.Len(); got != expCount {
		bt.f.T.Errorf(`Got %d function infos but expected %d.`, got, expCount)
		for i, inst := range bt.pkgInfo.funcInstInfos.Keys() {
			bt.f.T.Logf(`  %d. %q`, i+1, inst.TypeString())
		}
	}
}

func (bt *blockingTest) assertBlocking(funcName string) {
	if !bt.isTypesFuncBlocking(funcName) {
		bt.f.T.Errorf(`Got %q as not blocking but expected it to be blocking.`, funcName)
	}
}

func (bt *blockingTest) assertNotBlocking(funcName string) {
	if bt.isTypesFuncBlocking(funcName) {
		bt.f.T.Errorf(`Got %q as blocking but expected it to be not blocking.`, funcName)
	}
}

func getFuncDeclName(fd *ast.FuncDecl) string {
	name := fd.Name.Name
	if fd.Recv != nil && len(fd.Recv.List) == 1 && fd.Recv.List[0].Type != nil {
		typ := fd.Recv.List[0].Type
		if p, ok := typ.(*ast.StarExpr); ok {
			typ = p.X
		}
		if id, ok := typ.(*ast.Ident); ok {
			name = id.Name + `.` + name
		}
	}
	return name
}

func (bt *blockingTest) isTypesFuncBlocking(funcName string) bool {
	var decl *ast.FuncDecl
	ast.Inspect(bt.file, func(n ast.Node) bool {
		if f, ok := n.(*ast.FuncDecl); ok && getFuncDeclName(f) == funcName {
			decl = f
			return false
		}
		return decl == nil
	})

	if decl == nil {
		bt.f.T.Fatalf(`Declaration of %q is not found in the AST.`, funcName)
	}

	blockingType, ok := bt.pkgInfo.Defs[decl.Name]
	if !ok {
		bt.f.T.Fatalf(`No function declaration found for %q.`, decl.Name)
	}

	inst := typeparams.Instance{Object: blockingType.(*types.Func)}
	return bt.pkgInfo.IsBlocking(inst)
}

func (bt *blockingTest) assertBlockingLit(lineNo int) {
	if !bt.isFuncLitBlocking(lineNo) {
		bt.f.T.Errorf(`Got FuncLit at line %d as not blocking but expected it to be blocking.`, lineNo)
	}
}

func (bt *blockingTest) assertNotBlockingLit(lineNo int) {
	if bt.isFuncLitBlocking(lineNo) {
		bt.f.T.Errorf(`Got FuncLit at line %d as blocking but expected it to be not blocking.`, lineNo)
	}
}

func (bt *blockingTest) getFuncLitLineNo(fl *ast.FuncLit) int {
	return bt.f.FileSet.Position(fl.Pos()).Line
}

func (bt *blockingTest) isFuncLitBlocking(lineNo int) bool {
	var fnLit *ast.FuncLit
	ast.Inspect(bt.file, func(n ast.Node) bool {
		if fl, ok := n.(*ast.FuncLit); ok && bt.getFuncLitLineNo(fl) == lineNo {
			fnLit = fl
			return false
		}
		return fnLit == nil
	})

	if fnLit == nil {
		bt.f.T.Fatalf(`FuncLit found on line %d not found in the AST.`, lineNo)
	}
	return bt.pkgInfo.FuncLitInfo(fnLit).HasBlocking()
}

func (bt *blockingTest) assertBlockingInst(instanceStr string) {
	if !bt.isFuncInstBlocking(instanceStr) {
		bt.f.T.Errorf(`Got function instance of %q as not blocking but expected it to be blocking.`, instanceStr)
	}
}

func (bt *blockingTest) assertNotBlockingInst(instanceStr string) {
	if bt.isFuncInstBlocking(instanceStr) {
		bt.f.T.Errorf(`Got function instance of %q as blocking but expected it to be not blocking.`, instanceStr)
	}
}

func (bt *blockingTest) isFuncInstBlocking(instanceStr string) bool {
	instances := bt.pkgInfo.funcInstInfos.Keys()
	for _, inst := range instances {
		if inst.TypeString() == instanceStr {
			return bt.pkgInfo.FuncInfo(inst).HasBlocking()
		}
	}
	bt.f.T.Logf(`Function instances found in package info:`)
	for i, inst := range instances {
		bt.f.T.Logf(`  %d. %s`, i+1, inst.TypeString())
	}
	bt.f.T.Fatalf(`No function instance found for %q in package info.`, instanceStr)
	return false
}
