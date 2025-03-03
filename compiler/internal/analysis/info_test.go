package analysis

import (
	"fmt"
	"go/ast"
	"go/types"
	"sort"
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
	bt.assertBlockingLit(4, ``)

	bt.assertBlocking(`blocking`)
	bt.assertNotBlockingLit(10, ``)
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
	bt.assertBlockingLit(4, ``)

	bt.assertBlocking(`blockingArg`)
	bt.assertNotBlockingLit(10, ``)

	bt.assertNotBlocking(`notBlocking`)
	bt.assertNotBlockingLit(16, ``)
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
	bt.assertFuncInstCount(5)
	bt.assertFuncLitCount(0)

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
	bt.assertBlockingLit(4, ``)

	bt.assertBlocking(`blockingArg`)
	bt.assertNotBlockingLit(11, ``)

	bt.assertNotBlocking(`notBlocking`)
	bt.assertNotBlockingLit(18, ``)
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
			return 42 // line 13
		}

		func blockingArg(c chan bool) int {
			defer nonBlockingPrint(<-c)
			return 42 // line 18
		}

		func notBlocking(c chan bool) int {
			defer nonBlockingPrint(true)
			return 42 // line 23
		}`)
	bt.assertBlocking(`blockingPrint`)
	bt.assertNotBlocking(`nonBlockingPrint`)

	bt.assertBlocking(`blockingBody`)
	bt.assertBlockingReturn(13, ``)

	bt.assertBlocking(`blockingArg`)
	// The defer is non-blocking so the return is not blocking
	// even though the function is blocking.
	bt.assertNotBlockingReturn(18, ``)

	bt.assertNotBlocking(`notBlocking`)
	bt.assertNotBlockingReturn(23, ``)
}

func TestBlocking_Defers_WithMultipleReturns(t *testing.T) {
	bt := newBlockingTest(t,
		`package test

		func foo(c chan int) bool {
			defer func() { // line 4
				if r := recover(); r != nil {
					println("Error", r)
				}
			}()

			if c == nil {
				return false // line 11
			}

			defer func(v int) { // line 14
				println(v)
			}(<-c)

			value := <-c
			if value < 0 {
				return false // line 20
			}
			
			if value > 0 {
				defer func() { // line 24
					println(<-c)
				}()

				return false // line 28
			}

			return true // line 31
		}`)
	bt.assertBlocking(`foo`)
	bt.assertNotBlockingLit(4, ``)
	// Early escape from function without blocking defers is not blocking.
	bt.assertNotBlockingReturn(11, ``)
	bt.assertNotBlockingLit(14, ``)
	// Function has had blocking by this point but no blocking defers yet.
	bt.assertNotBlockingReturn(20, ``)
	bt.assertBlockingLit(24, ``)
	// The return is blocking because of a blocking defer.
	bt.assertBlockingReturn(28, ``)
	// Technically the return on line 31 is not blocking since the defer that
	// is blocking can only exit through the return on line 28, but it would be
	// difficult to determine which defers would only affect certain returns
	// without doing full control flow analysis.
	//
	// TODO(grantnelson-wf): We could fix this at some point by keeping track
	// of which flow control statements (e.g. if-statements) are terminating
	// or not. Any defers added in a terminating control flow would not
	// propagate to returns that are not in that block.
	//
	// For now we simply build up the list of defers as we go making
	// the return on line 31 also blocking.
	bt.assertBlockingReturn(31, ``)
}

func TestBlocking_Defers_WithReturnsAndDefaultBlocking(t *testing.T) {
	bt := newBlockingTest(t,
		`package test

		type foo struct {}
		func (f foo) Bar() {
			println("foo")
		}

		type stringer interface {
			Bar()
		}

		var fb = foo{}.Bar

		func deferInterfaceCall() bool {
			var s stringer = foo{}
			defer s.Bar()
			return true // line 17
		}

		func deferVarCall() bool {
			defer fb()
			return true // line 22
		}

		func deferLocalVarCall() bool {
			fp := foo{}.Bar
			defer fp()
			return true // line 28
		}

		func deferMethodExpressionCall() bool {
			fp := foo.Bar
			defer fp(foo{})
			return true // line 34
		}

		func deferSlicedFuncCall() bool {
			s := []func() { fb, foo{}.Bar }
			defer s[0]()
			return true // line 40
		}

		func deferMappedFuncCall() bool {
			m := map[string]func() {
				"fb": fb,
				"fNew": foo{}.Bar,
			}
			defer m["fb"]()
			return true // line 49
		}`)

	bt.assertFuncInstCount(7)
	bt.assertNotBlocking(`foo.Bar`)

	// None of these are actually blocking but we treat them like they are
	// because the defers invoke functions via interfaces and function pointers.
	bt.assertBlocking(`deferInterfaceCall`)
	bt.assertBlocking(`deferVarCall`)
	bt.assertBlocking(`deferLocalVarCall`)
	bt.assertBlocking(`deferMethodExpressionCall`)
	bt.assertBlocking(`deferSlicedFuncCall`)
	bt.assertBlocking(`deferMappedFuncCall`)

	// All of these returns are blocking because they have blocking defers.
	bt.assertBlockingReturn(17, ``)
	bt.assertBlockingReturn(22, ``)
	bt.assertBlockingReturn(28, ``)
	bt.assertBlockingReturn(34, ``)
	bt.assertBlockingReturn(40, ``)
	bt.assertBlockingReturn(49, ``)
}

func TestBlocking_Defers_WithReturnsAndDeferBuiltin(t *testing.T) {
	bt := newBlockingTest(t,
		`package test
		
		type strSet map[string]bool

		func deferBuiltinCall() strSet {
			m := strSet{
				"foo": true,
			}
			defer delete(m, "foo")
			return m // line 10
		}`)

	bt.assertFuncInstCount(1)
	bt.assertNotBlocking(`deferBuiltinCall`)
	bt.assertNotBlockingReturn(10, ``)
}

func TestBlocking_Defers_WithReturnsInLoops(t *testing.T) {
	// These are example of where a defer can affect the return that
	// occurs prior to the defer in the function body.
	bt := newBlockingTest(t,
		`package test

		func blocking(c chan int) {
			println(<-c)
		}

		func deferInForLoop(c chan int) bool {
			i := 1000
			for {
				i--
				if i <= 0 {
					return true // line 12
				}
				defer blocking(c)
			}
		}

		func deferInForLoopReturnAfter(c chan int) bool {
			for i := 1000; i > 0; i-- {
				defer blocking(c)
			}
			return true // line 22
		}

		func deferInNamedForLoop(c chan int) bool {
			i := 1000
		Start:
			for {
				i--
				if i <= 0 {
					return true // line 31
				}
				defer blocking(c)
				continue Start
			}
		}

		func deferInNamedForLoopReturnAfter(c chan int) bool {
		Start:
			for i := 1000; i > 0; i-- {
				defer blocking(c)
				continue Start
			}
			return true // line 44
		}

		func deferInGotoLoop(c chan int) bool {
			i := 1000
		Start:
			i--
			if i <= 0 {
				return true // line 52
			}
			defer blocking(c)
			goto Start
		}

		func deferInGotoLoopReturnAfter(c chan int) bool {
			i := 1000
		Start:
			defer blocking(c)
			i--
			if i > 0 {
				goto Start
			}
			return true // line 66
		}

		func deferInRangeLoop(c chan int) bool {
			s := []int{1, 2, 3}
			for i := range s {
				if i > 3 {
					return true // line 73
				}
				defer blocking(c)
			}
			return false // line 77
		}`)

	bt.assertFuncInstCount(8)
	bt.assertBlocking(`blocking`)
	bt.assertBlocking(`deferInForLoop`)
	bt.assertBlocking(`deferInForLoopReturnAfter`)
	bt.assertBlocking(`deferInNamedForLoop`)
	bt.assertBlocking(`deferInNamedForLoopReturnAfter`)
	bt.assertBlocking(`deferInGotoLoop`)
	bt.assertBlocking(`deferInGotoLoopReturnAfter`)
	bt.assertBlocking(`deferInRangeLoop`)
	// When the following 2 returns are defined there are no defers, however,
	// because of the loop, the blocking defers defined after the return will
	// block the returns.
	bt.assertBlockingReturn(12, ``)
	bt.assertBlockingReturn(22, ``)
	bt.assertBlockingReturn(31, ``)
	bt.assertBlockingReturn(44, ``)
	bt.assertBlockingReturn(52, ``)
	bt.assertBlockingReturn(66, ``)
	bt.assertBlockingReturn(73, ``)
	bt.assertBlockingReturn(77, ``)
}

func TestBlocking_Defers_WithReturnsInLoopsInLoops(t *testing.T) {
	// These are example of where a defer can affect the return that
	// occurs prior to the defer in the function body.
	bt := newBlockingTest(t,
		`package test

		func blocking(c chan int) {
			println(<-c)
		}

		func forLoopTheLoop(c chan int) bool {
			if c == nil {
				return false // line 9
			}
			for i := 0; i < 10; i++ {
				if i > 3 {
					return true // line 13
				}
				for j := 0; j < 10; j++ {
					if j > 3 {
						return true // line 17
					}
					defer blocking(c)
					if j > 2 {
						return false // line 21
					}
				}
				if i > 2 {
					return false // line 25
				}
			}
			return false // line 28
		}

		func rangeLoopTheLoop(c chan int) bool {
			data := []int{1, 2, 3}
			for i := range data {
				for j := range data {
					if i + j > 3 {
						return true // line 36
					}
				}
				defer blocking(c)
			}
			return false // line 41
		}

		func noopThenLoop(c chan int) bool {
			data := []int{1, 2, 3}
			for i := range data {
				if i > 13 {
					return true // line 48
				}
				defer func() { println("hi") }()
			}
			for i := range data {
				if i > 3 {
					return true // line 54
				}
				defer blocking(c)
			}
			return false // line 58
		}`)

	bt.assertFuncInstCount(4)
	bt.assertBlocking(`blocking`)
	bt.assertBlocking(`forLoopTheLoop`)
	bt.assertNotBlockingReturn(9, ``)
	bt.assertBlockingReturn(13, ``)
	bt.assertBlockingReturn(17, ``)
	bt.assertBlockingReturn(21, ``)
	bt.assertBlockingReturn(25, ``)
	bt.assertBlockingReturn(28, ``)
	bt.assertBlocking(`rangeLoopTheLoop`)
	bt.assertBlockingReturn(36, ``)
	bt.assertBlockingReturn(41, ``)
	bt.assertBlocking(`noopThenLoop`)
	bt.assertNotBlockingReturn(48, ``)
	bt.assertBlockingReturn(54, ``)
	bt.assertBlockingReturn(58, ``)
}

func TestBlocking_Returns_WithoutDefers(t *testing.T) {
	bt := newBlockingTest(t,
		`package test

		func blocking(c chan bool) bool {
			return <-c // line 4
		}

		func blockingBeforeReturn(c chan bool) bool {
			v := <-c
			return v // line 9
		}

		func indirectlyBlocking(c chan bool) bool {
			return blocking(c) // line 13
		}

		func indirectlyBlockingBeforeReturn(c chan bool) bool {
			v := blocking(c)
			return v // line 18
		}

		func notBlocking(c chan bool) bool {
			return true // line 22
		}`)
	bt.assertBlocking(`blocking`)
	bt.assertBlockingReturn(4, ``)

	bt.assertBlocking(`blockingBeforeReturn`)
	bt.assertNotBlockingReturn(9, ``)

	bt.assertBlocking(`indirectlyBlocking`)
	bt.assertBlockingReturn(13, ``)

	bt.assertBlocking(`indirectlyBlockingBeforeReturn`)
	bt.assertNotBlockingReturn(18, ``)

	bt.assertNotBlocking(`notBlocking`)
	bt.assertNotBlockingReturn(22, ``)
}

func TestBlocking_Defers_WithReturnsInInstances(t *testing.T) {
	// This is an example of a deferred function literal inside of
	// an instance of a generic function affecting the return
	// differently based on the type arguments of the instance.
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
		func FooBaz[T Foo]() bool {
			defer func() { // line 17
				var foo T
				foo.Baz()
			}()
			return true // line 21
		}

		func main() {
			FooBaz[BazBlocker]()
			FooBaz[BazNotBlocker]()
		}`)

	bt.assertFuncInstCount(5)
	bt.assertBlocking(`BazBlocker.Baz`)
	bt.assertNotBlocking(`BazNotBlocker.Baz`)
	bt.assertBlockingInst(`pkg/test.FooBaz<pkg/test.BazBlocker>`)
	bt.assertNotBlockingInst(`pkg/test.FooBaz<pkg/test.BazNotBlocker>`)
	bt.assertBlocking(`main`)

	bt.assertFuncLitCount(2)
	bt.assertBlockingLit(17, `pkg/test.BazBlocker`)
	bt.assertNotBlockingLit(17, `pkg/test.BazNotBlocker`)

	bt.assertBlockingReturn(21, `pkg/test.BazBlocker`)
	bt.assertNotBlockingReturn(21, `pkg/test.BazNotBlocker`)
}

func TestBlocking_Defers_WithReturnsAndOtherPackages(t *testing.T) {
	otherSrc := `package other

		func Blocking() {
			c := make(chan int)
			println(<-c)
		}

		func NotBlocking() {
			println("Hello")
		}`

	testSrc := `package test

		import "pkg/other"

		func deferOtherBlocking() bool {
			defer other.Blocking()
			return true // line 7
		}
		
		func deferOtherNotBlocking() bool {
			defer other.NotBlocking()
			return true // line 12
		}`

	bt := newBlockingTestWithOtherPackage(t, testSrc, otherSrc)

	bt.assertBlocking(`deferOtherBlocking`)
	bt.assertBlockingReturn(7, ``)

	bt.assertNotBlocking(`deferOtherNotBlocking`)
	bt.assertNotBlockingReturn(12, ``)
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
	bt.assertBlockingLit(9, ``)

	bt.assertBlocking(`directlyBlocking`)
	bt.assertBlockingLit(13, ``)

	bt.assertNotBlocking(`notBlocking`)
	bt.assertNotBlockingLit(20, ``)
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

	bt.assertBlockingInst(`pkg/test.blocking<int>`)
	bt.assertBlocking(`bInt`)
	bt.assertNotBlockingInst(`pkg/test.notBlocking<uint>`)
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

	bt.assertBlockingInst(`pkg/test.blocking<string, int, map[string]int>`)
	bt.assertBlocking(`bInt`)
	bt.assertNotBlockingInst(`pkg/test.notBlocking<string, uint, map[string]uint>`)
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
	bt.assertNotBlockingLit(3, ``)
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
	bt.assertBlockingInst(`pkg/test.FooBaz<pkg/test.BazBlocker>`)

	bt.assertNotBlocking(`BazNotBlocker.Baz`)
	bt.assertNotBlocking(`notBlockingViaExplicit`)
	bt.assertNotBlocking(`notBlockingViaImplicit`)
	bt.assertNotBlockingInst(`pkg/test.FooBaz<pkg/test.BazNotBlocker>`)
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
	bt.assertNotBlockingInst(`pkg/test.Foo<map[int]int>`)
	bt.assertNotBlockingInst(`pkg/test.Foo<map[int]string>`)
	bt.assertNotBlockingInst(`pkg/test.Bar<int, int, map[int]int>`)
	bt.assertNotBlockingInst(`pkg/test.Bar<int, string, map[int]string>`)
	bt.assertNotBlockingInst(`pkg/test.Baz<int, []int>`)
	bt.assertNotBlockingInst(`pkg/test.Baz<string, []string>`)
}

func TestBlocking_UnusedGenericFunctions(t *testing.T) {
	// Checking that the type parameters are being propagated down into callee.
	// This is based off of go1.19.13/test/typeparam/orderedmap.go
	bt := newBlockingTest(t,
		`package test

		type node[K, V any] struct {
			key         K
			val         V
			left, right *node[K, V]
		}

		type Tree[K, V any] struct {
			root *node[K, V]
			eq   func(K, K) bool
		}

		func New[K, V any](eq func(K, K) bool) *Tree[K, V] {
			return &Tree[K, V]{eq: eq}
		}

		func NewStrKey[K ~string, V any]() *Tree[K, V] { // unused
			return New[K, V](func(k1, k2 K) bool {
				return string(k1) == string(k2)
			})
		}

		func NewStrStr[V any]() *Tree[string, V] { // unused
			return NewStrKey[string, V]()
		}

		func main() {
			t := New[int, string](func(k1, k2 int) bool {
				return k1 == k2
			})
			println(t)
		}`)
	bt.assertFuncInstCount(2)
	// Notice that `NewStrKey` and `NewStrStr` are not called so doesn't have
	// any known instances and therefore they don't have any FuncInfos.
	bt.assertNotBlockingInst(`pkg/test.New<int, string>`)
	bt.assertNotBlocking(`main`)
}

func TestBlocking_LitInstanceCalls(t *testing.T) {
	// Literals defined inside a generic function must inherit the
	// type arguments (resolver) of the enclosing instance it is defined in
	// so that things like calls to other generic functions create the
	// call to the correct concrete instance.
	bt := newBlockingTest(t,
		`package test

		func foo[T any](x T) {
			println(x)
		}

		func bar[T any](x T) {
			f := func(v T) { // line 8
				foo[T](v)
			}
			f(x)
		}

		func main() {
			bar[int](42)
			bar[float64](3.14)
		}`)
	bt.assertFuncInstCount(5)

	bt.assertNotBlockingInst(`pkg/test.foo<int>`)
	bt.assertNotBlockingInst(`pkg/test.foo<float64>`)
	bt.assertNotBlockingLit(8, `int`)
	bt.assertNotBlockingLit(8, `float64`)
	// The following are blocking because the function literal call.
	bt.assertBlockingInst(`pkg/test.bar<int>`)
	bt.assertBlockingInst(`pkg/test.bar<float64>`)
}

func TestBlocking_BlockingLitInstance(t *testing.T) {
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
		func FooBaz[T Foo](foo T) func() {
			return func() { // line 17
				foo.Baz()
			}
		}

		func main() {
			_ = FooBaz(BazBlocker{})
			_ = FooBaz(BazNotBlocker{})
		}`)
	bt.assertFuncInstCount(5)

	bt.assertBlocking(`BazBlocker.Baz`)
	// THe following is not blocking because the function literal is not called.
	bt.assertNotBlockingInst(`pkg/test.FooBaz<pkg/test.BazBlocker>`)
	bt.assertBlockingLit(17, `pkg/test.BazBlocker`)

	bt.assertNotBlocking(`BazNotBlocker.Baz`)
	bt.assertNotBlockingInst(`pkg/test.FooBaz<pkg/test.BazNotBlocker>`)
	bt.assertNotBlockingLit(17, `pkg/test.BazNotBlocker`)
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
	bt.assertBlockingInst(`pkg/test.FooBaz.ByMethodExpression<pkg/test.BazBlocker>`)
	bt.assertBlockingInst(`pkg/test.FooBaz.ByInstance<pkg/test.BazBlocker>`)
	bt.assertBlocking(`blocking`)

	bt.assertNotBlocking(`BazNotBlocker.Baz`)
	bt.assertNotBlockingInst(`pkg/test.FooBaz.ByMethodExpression<pkg/test.BazNotBlocker>`)
	bt.assertNotBlockingInst(`pkg/test.FooBaz.ByInstance<pkg/test.BazNotBlocker>`)
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
	// This is tests propagation of information across package boundaries.
	// `FooBaz` has no instances in it until it is referenced in the `test` package.
	// That instance information needs to propagate back across the package
	// boundary to the `other` package. The information for `BazBlocker` and
	// `BazNotBlocker` is propagated back to `FooBaz[BazBlocker]` and
	// `FooBaz[BazNotBlocker]`. That information is then propagated forward
	// to the `blocking` and `notBlocking` functions in the `test` package.

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
	tContext := types.NewContext()
	tc := typeparams.Collector{
		TContext:  tContext,
		Info:      f.Info,
		Instances: &typeparams.PackageInstanceSets{},
	}

	file := f.Parse(`test.go`, src)
	testInfo, testPkg := f.Check(`pkg/test`, file)
	tc.Scan(testPkg, file)

	getImportInfo := func(path string) (*Info, error) {
		return nil, fmt.Errorf(`getImportInfo should not be called in this test, called with %v`, path)
	}
	pkgInfo := AnalyzePkg([]*ast.File{file}, f.FileSet, testInfo, tContext, testPkg, tc.Instances, getImportInfo)
	PropagateAnalysis([]*Info{pkgInfo})

	return &blockingTest{
		f:       f,
		file:    file,
		pkgInfo: pkgInfo,
	}
}

func newBlockingTestWithOtherPackage(t *testing.T, testSrc string, otherSrc string) *blockingTest {
	f := srctesting.New(t)
	tContext := types.NewContext()
	tc := typeparams.Collector{
		TContext:  tContext,
		Info:      f.Info,
		Instances: &typeparams.PackageInstanceSets{},
	}

	pkgInfo := map[string]*Info{}
	getImportInfo := func(path string) (*Info, error) {
		if info, ok := pkgInfo[path]; ok {
			return info, nil
		}
		return nil, fmt.Errorf(`unexpected package in getImportInfo for %v`, path)
	}

	otherFile := f.Parse(`other.go`, otherSrc)
	_, otherPkg := f.Check(`pkg/other`, otherFile)
	tc.Scan(otherPkg, otherFile)

	testFile := f.Parse(`test.go`, testSrc)
	_, testPkg := f.Check(`pkg/test`, testFile)
	tc.Scan(testPkg, testFile)

	otherPkgInfo := AnalyzePkg([]*ast.File{otherFile}, f.FileSet, f.Info, tContext, otherPkg, tc.Instances, getImportInfo)
	pkgInfo[otherPkg.Path()] = otherPkgInfo

	testPkgInfo := AnalyzePkg([]*ast.File{testFile}, f.FileSet, f.Info, tContext, testPkg, tc.Instances, getImportInfo)
	pkgInfo[testPkg.Path()] = testPkgInfo

	PropagateAnalysis([]*Info{otherPkgInfo, testPkgInfo})

	return &blockingTest{
		f:       f,
		file:    testFile,
		pkgInfo: testPkgInfo,
	}
}

func (bt *blockingTest) assertFuncInstCount(expCount int) {
	bt.f.T.Helper()
	if got := bt.pkgInfo.funcInstInfos.Len(); got != expCount {
		bt.f.T.Errorf(`Got %d function instance infos but expected %d.`, got, expCount)
		for i, inst := range bt.pkgInfo.funcInstInfos.Keys() {
			bt.f.T.Logf(`  %d. %q`, i+1, inst.String())
		}
	}
}

func (bt *blockingTest) assertFuncLitCount(expCount int) {
	bt.f.T.Helper()
	got := 0
	for _, fis := range bt.pkgInfo.funcLitInfos {
		got += len(fis)
	}
	if got != expCount {
		bt.f.T.Errorf(`Got %d function literal infos but expected %d.`, got, expCount)

		lits := make([]string, 0, len(bt.pkgInfo.funcLitInfos))
		for fl, fis := range bt.pkgInfo.funcLitInfos {
			pos := bt.f.FileSet.Position(fl.Pos()).String()
			for _, fi := range fis {
				lits = append(lits, pos+`<`+fi.typeArgs.String()+`>`)
			}
		}
		sort.Strings(lits)
		for i := range lits {
			bt.f.T.Logf(`  %d. %q`, i+1, lits[i])
		}
	}
}

func (bt *blockingTest) assertBlocking(funcName string) {
	bt.f.T.Helper()
	if !bt.isTypesFuncBlocking(funcName) {
		bt.f.T.Errorf(`Got %q as not blocking but expected it to be blocking.`, funcName)
	}
}

func (bt *blockingTest) assertNotBlocking(funcName string) {
	bt.f.T.Helper()
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
	bt.f.T.Helper()
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

func (bt *blockingTest) assertBlockingLit(lineNo int, typeArgsStr string) {
	bt.f.T.Helper()
	if !bt.isFuncLitBlocking(lineNo, typeArgsStr) {
		bt.f.T.Errorf(`Got FuncLit at line %d with type args %q as not blocking but expected it to be blocking.`, lineNo, typeArgsStr)
	}
}

func (bt *blockingTest) assertNotBlockingLit(lineNo int, typeArgsStr string) {
	bt.f.T.Helper()
	if bt.isFuncLitBlocking(lineNo, typeArgsStr) {
		bt.f.T.Errorf(`Got FuncLit at line %d with type args %q as blocking but expected it to be not blocking.`, lineNo, typeArgsStr)
	}
}

func (bt *blockingTest) isFuncLitBlocking(lineNo int, typeArgsStr string) bool {
	bt.f.T.Helper()
	fnLit := srctesting.GetNodeAtLineNo[*ast.FuncLit](bt.file, bt.f.FileSet, lineNo)
	if fnLit == nil {
		bt.f.T.Fatalf(`FuncLit on line %d not found in the AST.`, lineNo)
	}

	fis, ok := bt.pkgInfo.funcLitInfos[fnLit]
	if !ok {
		bt.f.T.Fatalf(`No FuncInfo found for FuncLit at line %d.`, lineNo)
	}

	for _, fi := range fis {
		if fi.typeArgs.String() == typeArgsStr {
			return fi.IsBlocking()
		}
	}

	bt.f.T.Logf("FuncList instances:")
	for i, fi := range fis {
		bt.f.T.Logf("\t%d. %q\n", i+1, fi.typeArgs.String())
	}
	bt.f.T.Fatalf(`No FuncInfo found for FuncLit at line %d with type args %q.`, lineNo, typeArgsStr)
	return false
}

func (bt *blockingTest) assertBlockingInst(instanceStr string) {
	bt.f.T.Helper()
	if !bt.isFuncInstBlocking(instanceStr) {
		bt.f.T.Errorf(`Got function instance of %q as not blocking but expected it to be blocking.`, instanceStr)
	}
}

func (bt *blockingTest) assertNotBlockingInst(instanceStr string) {
	bt.f.T.Helper()
	if bt.isFuncInstBlocking(instanceStr) {
		bt.f.T.Errorf(`Got function instance of %q as blocking but expected it to be not blocking.`, instanceStr)
	}
}

func (bt *blockingTest) isFuncInstBlocking(instanceStr string) bool {
	bt.f.T.Helper()
	instances := bt.pkgInfo.funcInstInfos.Keys()
	for _, inst := range instances {
		if inst.String() == instanceStr {
			return bt.pkgInfo.FuncInfo(inst).IsBlocking()
		}
	}
	bt.f.T.Logf(`Function instances found in package info:`)
	for i, inst := range instances {
		bt.f.T.Logf("\t%d. %s", i+1, inst.String())
	}
	bt.f.T.Fatalf(`No function instance found for %q in package info.`, instanceStr)
	return false
}

func (bt *blockingTest) assertBlockingReturn(lineNo int, typeArgsStr string) {
	bt.f.T.Helper()
	if !bt.isReturnBlocking(lineNo, typeArgsStr) {
		bt.f.T.Errorf(`Got return at line %d (%q) as not blocking but expected it to be blocking.`, lineNo, typeArgsStr)
	}
}

func (bt *blockingTest) assertNotBlockingReturn(lineNo int, typeArgsStr string) {
	bt.f.T.Helper()
	if bt.isReturnBlocking(lineNo, typeArgsStr) {
		bt.f.T.Errorf(`Got return at line %d (%q) as blocking but expected it to be not blocking.`, lineNo, typeArgsStr)
	}
}

func (bt *blockingTest) isReturnBlocking(lineNo int, typeArgsStr string) bool {
	bt.f.T.Helper()
	ret := srctesting.GetNodeAtLineNo[*ast.ReturnStmt](bt.file, bt.f.FileSet, lineNo)
	if ret == nil {
		bt.f.T.Fatalf(`ReturnStmt on line %d not found in the AST.`, lineNo)
	}

	foundInfo := []*FuncInfo{}
	for _, info := range bt.pkgInfo.allInfos {
		for _, rs := range info.returnStmts {
			if rs.analyzeStack[len(rs.analyzeStack)-1] == ret {
				if info.typeArgs.String() == typeArgsStr {
					// Found info that matches the type args and
					// has the return statement so return the blocking value.
					return info.Blocking[ret]
				}

				// Wrong instance, record for error message in the case
				// that the correct one instance is not found.
				foundInfo = append(foundInfo, info)
				break
			}
		}
	}

	bt.f.T.Logf("FuncInfo instances with ReturnStmt at line %d:", lineNo)
	for i, info := range foundInfo {
		bt.f.T.Logf("\t%d. %q\n", i+1, info.typeArgs.String())
	}
	bt.f.T.Fatalf(`No FuncInfo found for ReturnStmt at line %d with type args %q.`, lineNo, typeArgsStr)
	return false
}
