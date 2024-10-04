package analysis

import (
	"go/ast"
	"go/types"
	"testing"

	"github.com/gopherjs/gopherjs/internal/srctesting"
)

func TestBlockingSimplePrint(t *testing.T) {
	blockingTest(t, blockingTestArgs{
		src: `package test
			func notBlocking() {
				println("hi")
			}`,
		notBlocking: []string{`notBlocking`},
	})
}

func TestBlockingChannels(t *testing.T) {
	blockingTest(t, blockingTestArgs{
		src: `package test
			func readFromChannel(c chan bool) {
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
			}`,
		blocking:    []string{`readFromChannel`, `sendToChannel`, `rangeOnChannel`},
		notBlocking: []string{`rangeOnSlice`},
	})
}

func TestBlockingSelects(t *testing.T) {
	blockingTest(t, blockingTestArgs{
		src: `package test
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
			}`,
		blocking:    []string{`selectReadWithoutDefault`, `selectSendWithoutDefault`},
		notBlocking: []string{`selectReadWithDefault`, `selectSendWithDefault`},
	})
}

func TestBlockingGos(t *testing.T) {
	blockingTest(t, blockingTestArgs{
		src: `package test
			func notBlocking(c chan bool) {
				go func(c chan bool) {
					println(<-c)
				}(c)
			}

			func blocking(c chan bool) {
				go func(v bool) {
					println(v)
				}(<-c)
			}`,
		blocking:    []string{`blocking`},
		notBlocking: []string{`notBlocking`},
	})
}

func TestBlockingDefers(t *testing.T) {
	blockingTest(t, blockingTestArgs{
		src: `package test
			func blockingBody(c chan bool) {
				defer func(c chan bool) {
					println(<-c)
				}(c)
			}

			func blockingArg(c chan bool) {
				defer func(v bool) {
					println(v)
				}(<-c)
			}

			func notBlocking(c chan bool) {
				defer func(v bool) {
					println(v)
				}(true)
			}`,
		blocking:    []string{`blockingBody`, `blockingArg`},
		notBlocking: []string{`notBlocking`},
	})
}

func TestBlockingReturns(t *testing.T) {
	blockingTest(t, blockingTestArgs{
		src: `package test
			func blocking(c chan bool) bool {
				return <-c
			}

			func indirectlyBlocking(c chan bool) bool {
				return blocking(c)
			}

			func notBlocking(c chan bool) bool {
				return true
			}`,
		blocking:    []string{`blocking`, `indirectlyBlocking`},
		notBlocking: []string{`notBlocking`},
	})
}

func TestBlockingFunctionLiteral(t *testing.T) {
	// See: https://github.com/gopherjs/gopherjs/issues/955.
	blockingTest(t, blockingTestArgs{
		src: `package test

			func blocking() {
				c := make(chan bool)
				<-c
			}

			func indirectlyBlocking() {
				func() { blocking() }()
			}

			func directlyBlocking() {
				func() {
					c := make(chan bool)
					<-c
				}()
			}

			func notBlocking() {
				func() { println() } ()
			}`,
		blocking:    []string{`blocking`, `indirectlyBlocking`, `directlyBlocking`},
		notBlocking: []string{`notBlocking`},
	})
}

func TestBlockingLinkedFunction(t *testing.T) {
	blockingTest(t, blockingTestArgs{
		src: `package test

			// linked to some other function
			func blocking()

			func indirectlyBlocking() {
				blocking()
			}`,
		blocking: []string{`blocking`, `indirectlyBlocking`},
	})
}

func TestBlockingInstanceWithSingleTypeArgument(t *testing.T) {
	blockingTest(t, blockingTestArgs{
		src: `package test
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
			}`,
		blocking:    []string{`blocking`, `bInt`},
		notBlocking: []string{`notBlocking`, `nbUint`},
	})
}

func TestBlockingInstanceWithMultipleTypeArguments(t *testing.T) {
	blockingTest(t, blockingTestArgs{
		src: `package test
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
			}`,
		blocking:    []string{`blocking`, `bInt`},
		notBlocking: []string{`notBlocking`, `nbUint`},
	})
}

func TestBlockingIndexedFromFunctionSlice(t *testing.T) {
	// This calls notBlocking but since the function pointers
	// are in the slice they will both be considered as blocking.
	// This is just checking that the analysis can tell between
	// indexing and instantiation of a generic.
	blockingTest(t, blockingTestArgs{
		src: `package test
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
			}`,
		blocking:    []string{`blocking`, `indexer`},
		notBlocking: []string{`notBlocking`},
	})
}

func TestBlockingCastingToAnInterfaceInstance(t *testing.T) {
	// This checks that casting to an instance type is treated as a
	// cast an not accidentally treated as a function call.
	blockingTest(t, blockingTestArgs{
		src: `package test
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
				}`,
		notBlocking: []string{`caster`},
	})
}

func TestBlockingCastingToAnInterface(t *testing.T) {
	// This checks of non-generic casting of type is treated as a
	// cast an not accidentally treated as a function call.
	blockingTest(t, blockingTestArgs{
		src: `package test
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
			}`,
		notBlocking: []string{`caster`},
	})
}

type blockingTestArgs struct {
	src         string
	blocking    []string
	notBlocking []string
}

func blockingTest(t *testing.T, test blockingTestArgs) {
	f := srctesting.New(t)

	file := f.Parse(`test.go`, test.src)
	typesInfo, typesPkg := f.Check(`pkg/test`, file)

	pkgInfo := AnalyzePkg([]*ast.File{file}, f.FileSet, typesInfo, typesPkg, func(f *types.Func) bool {
		panic(`isBlocking() should be never called for imported functions in this test.`)
	})

	for _, funcName := range test.blocking {
		assertBlocking(t, file, pkgInfo, funcName)
	}

	for _, funcName := range test.notBlocking {
		assertNotBlocking(t, file, pkgInfo, funcName)
	}
}

func assertBlocking(t *testing.T, file *ast.File, pkgInfo *Info, funcName string) {
	typesFunc := getTypesFunc(t, file, pkgInfo, funcName)
	if !pkgInfo.IsBlocking(typesFunc) {
		t.Errorf("Got: %q is not blocking. Want: %q is blocking.", typesFunc, typesFunc)
	}
}

func assertNotBlocking(t *testing.T, file *ast.File, pkgInfo *Info, funcName string) {
	typesFunc := getTypesFunc(t, file, pkgInfo, funcName)
	if pkgInfo.IsBlocking(typesFunc) {
		t.Errorf("Got: %q is blocking. Want: %q is not blocking.", typesFunc, typesFunc)
	}
}

func getTypesFunc(t *testing.T, file *ast.File, pkgInfo *Info, funcName string) *types.Func {
	obj := file.Scope.Lookup(funcName)
	if obj == nil {
		t.Fatalf("Declaration of %q is not found in the AST.", funcName)
	}
	decl, ok := obj.Decl.(*ast.FuncDecl)
	if !ok {
		t.Fatalf("Got: %q is %v. Want: a function declaration.", funcName, obj.Kind)
	}
	blockingType, ok := pkgInfo.Defs[decl.Name]
	if !ok {
		t.Fatalf("No type information is found for %v.", decl.Name)
	}
	return blockingType.(*types.Func)
}
