package analysis

import (
	"go/ast"
	"go/types"
	"testing"

	"github.com/gopherjs/gopherjs/internal/srctesting"
)

func TestBlockingSimple(t *testing.T) {
	bt := newBlockingTest(t,
		`package test

		func notBlocking() {
			println("hi")
		}`)
	bt.assertNotBlocking(`notBlocking`)
}

func TestBlockingRecursive(t *testing.T) {
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

func TestBlockingAlternatingRecursive(t *testing.T) {
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

func TestBlockingChannels(t *testing.T) {
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

func TestBlockingSelects(t *testing.T) {
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

func TestBlockingGos(t *testing.T) {
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

	// TODO: Add go with non-lit func

	bt.assertNotBlocking(`notBlocking`)
	bt.assertBlockingLit(4)

	bt.assertBlocking(`blocking`)
	bt.assertNotBlockingLit(10)
}

func TestBlockingDefers(t *testing.T) {
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

	// TODO: Add defer with non-lit func

	bt.assertBlocking(`blockingBody`)
	bt.assertBlockingLit(4)

	bt.assertBlocking(`blockingArg`)
	bt.assertNotBlockingLit(10)

	bt.assertNotBlocking(`notBlocking`)
	bt.assertNotBlockingLit(16)
}

func TestBlockingReturns(t *testing.T) {
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

func TestBlockingFunctionLiteral(t *testing.T) {
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

func TestBlockingLinkedFunction(t *testing.T) {
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

func TestBlockingInstanceWithSingleTypeArgument(t *testing.T) {
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
	bt.assertBlocking(`blocking`)
	bt.assertBlocking(`bInt`)
	bt.assertNotBlocking(`notBlocking`)
	bt.assertNotBlocking(`nbUint`)
}

func TestBlockingInstanceWithMultipleTypeArguments(t *testing.T) {
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
	bt.assertBlocking(`blocking`)
	bt.assertBlocking(`bInt`)
	bt.assertNotBlocking(`notBlocking`)
	bt.assertNotBlocking(`nbUint`)
}

func TestBlockingIndexedFromFunctionSlice(t *testing.T) {
	// This calls notBlocking but since the function pointers
	// are in the slice they will both be considered as blocking.
	// This is just checking that the analysis can tell between
	// indexing and instantiation of a generic.
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

func TestBlockingCastingToAnInterfaceInstance(t *testing.T) {
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

func TestBlockingCastingToAnInterface(t *testing.T) {
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

func TestBlockingInstantiationBlocking(t *testing.T) {
	// This checks that the instantiation of a generic function
	// is being used when checking for blocking.
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

		func blocking() {
			FooBaz(BazBlocker{c: make(chan bool)})
		}
		
		func notBlocking() {
			FooBaz(BazNotBlocker{})
		}`)
	bt.assertBlocking(`FooBaz`) // generic instantiation is blocking
	bt.assertBlocking(`BazBlocker.Baz`)
	bt.assertBlocking(`blocking`)
	bt.assertNotBlocking(`BazNotBlocker.Baz`)
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

	pkgInfo := AnalyzePkg([]*ast.File{file}, f.FileSet, typesInfo, typesPkg, func(f *types.Func) bool {
		panic(`isBlocking() should be never called for imported functions in this test.`)
	})

	return &blockingTest{
		f:       f,
		file:    file,
		pkgInfo: pkgInfo,
	}
}

func (bt *blockingTest) assertBlocking(funcName string) {
	if !bt.isTypesFuncBlocking(funcName) {
		bt.f.T.Errorf(`Got: %q is not blocking. Want: %q is blocking.`, funcName, funcName)
	}
}

func (bt *blockingTest) assertNotBlocking(funcName string) {
	if bt.isTypesFuncBlocking(funcName) {
		bt.f.T.Errorf(`Got: %q is blocking. Want: %q is not blocking.`, funcName, funcName)
	}
}

func (bt *blockingTest) isTypesFuncBlocking(funcName string) bool {
	var decl *ast.FuncDecl
	ast.Inspect(bt.file, func(n ast.Node) bool {
		if f, ok := n.(*ast.FuncDecl); ok {
			name := f.Name.Name
			if f.Recv != nil {
				name = f.Recv.List[0].Type.(*ast.Ident).Name + `.` + name
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
		bt.f.T.Fatalf(`No type information is found for %v.`, decl.Name)
	}
	return bt.pkgInfo.IsBlocking(blockingType.(*types.Func))
}

func (bt *blockingTest) assertBlockingLit(lineNo int) {
	if !bt.isFuncLitBlocking(lineNo) {
		bt.f.T.Errorf(`Got: FuncLit at line %d is not blocking. Want: FuncLit at line %d is blocking.`, lineNo, lineNo)
	}
}

func (bt *blockingTest) assertNotBlockingLit(lineNo int) {
	if bt.isFuncLitBlocking(lineNo) {
		bt.f.T.Errorf(`Got: FuncLit at line %d is blocking. Want: FuncLit at line %d is not blocking.`, lineNo, lineNo)
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
	info, ok := bt.pkgInfo.FuncLitInfos[fnLit]
	if !ok {
		bt.f.T.Fatalf(`No type information is found for FuncLit at line %d.`, lineNo)
	}
	return len(info.Blocking) > 0
}
