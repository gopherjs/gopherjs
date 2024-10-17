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

func TestBlocking_Casting_InterfaceInstance(t *testing.T) {
	// This checks that casting to an instance type is treated as a
	// cast an not accidentally treated as a function call.
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
			b := Bar{"foo"}
			return Foo[string](b)
		}`)
	bt.assertNotBlocking(`caster`)
}

func TestBlocking_Casting_Interface(t *testing.T) {
	// This checks of non-generic casting of type is treated as a
	// cast an not accidentally treated as a function call.
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

func TestBlocking_MethodExpressions(t *testing.T) {
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
		func (fb FooBaz[T]) Baz() {
			var foo T
			foo.Baz()
		}

		func blocking() {
			FooBaz[BazBlocker].Baz(FooBaz[BazBlocker]{})
		}
		
		func notBlocking() {
			FooBaz[BazNotBlocker].Baz(FooBaz[BazNotBlocker]{})
		}`)
	bt.assertFuncInstCount(6)

	bt.assertBlocking(`BazBlocker.Baz`)
	bt.assertBlockingInst(`test.Baz[pkg/test.BazBlocker]`)
	bt.assertBlocking(`blocking`)

	bt.assertNotBlocking(`BazNotBlocker.Baz`)
	bt.assertNotBlockingInst(`test.Baz[pkg/test.BazNotBlocker]`)
	bt.assertNotBlocking(`notBlocking`)
}

type blockingTest struct {
	f       *srctesting.Fixture
	file    *ast.File
	pkgInfo *Info
}

func newBlockingTest(t *testing.T, src string) *blockingTest {
	f := srctesting.New(t)

	file := f.Parse(`test.go`, src)
	typesInfo, typesPkg := f.Check(`pkg/test`, file)

	tc := typeparams.Collector{
		TContext:  types.NewContext(),
		Info:      typesInfo,
		Instances: &typeparams.PackageInstanceSets{},
	}
	tc.Scan(typesPkg, file)

	pkgInfo := AnalyzePkg([]*ast.File{file}, f.FileSet, typesInfo, types.NewContext(), typesPkg, tc.Instances, func(f *types.Func) bool {
		panic(`isBlocking() should be never called for imported functions in this test.`)
	})

	return &blockingTest{
		f:       f,
		file:    file,
		pkgInfo: pkgInfo,
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

func (bt *blockingTest) isTypesFuncBlocking(funcName string) bool {
	var decl *ast.FuncDecl
	ast.Inspect(bt.file, func(n ast.Node) bool {
		if f, ok := n.(*ast.FuncDecl); ok {
			name := f.Name.Name
			if f.Recv != nil {
				if id, ok := f.Recv.List[0].Type.(*ast.Ident); ok {
					name = id.Name + `.` + name
				}
			}
			if name == funcName {
				decl = f
				return false
			}
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
	funcInfo := bt.pkgInfo.funcInstInfos.Get(inst)
	if funcInfo == nil {
		bt.f.T.Fatalf(`No function instance info found for %q in package info.`, funcName)
	}
	return funcInfo.HasBlocking()
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

func (bt *blockingTest) isFuncLitBlocking(lineNo int) bool {
	var fnLit *ast.FuncLit
	ast.Inspect(bt.file, func(n ast.Node) bool {
		if fl, ok := n.(*ast.FuncLit); ok {
			if bt.f.FileSet.Position(fl.Pos()).Line == lineNo {
				fnLit = fl
				return false
			}
		}
		return fnLit == nil
	})

	if fnLit == nil {
		bt.f.T.Fatalf(`FuncLit found on line %d not found in the AST.`, lineNo)
	}
	info, ok := bt.pkgInfo.funcLitInfos[fnLit]
	if !ok {
		bt.f.T.Fatalf(`No type information is found for FuncLit at line %d.`, lineNo)
	}
	return info.HasBlocking()
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
			funcInfo := bt.pkgInfo.funcInstInfos.Get(inst)
			if funcInfo == nil {
				bt.f.T.Fatalf(`No function instance info found for function instance %q in package info.`, instanceStr)
			}
			return funcInfo.HasBlocking()
		}
	}
	bt.f.T.Logf(`Function instances found in package info:`)
	for i, inst := range instances {
		bt.f.T.Logf(`  %d. %s`, i+1, inst.TypeString())
	}
	bt.f.T.Fatalf(`No function instance found for %q in package info.`, instanceStr)
	return false
}
