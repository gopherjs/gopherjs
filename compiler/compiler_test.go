package compiler

import (
	"bytes"
	"go/types"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/tools/go/packages"

	"github.com/gopherjs/gopherjs/compiler/internal/dce"
	"github.com/gopherjs/gopherjs/compiler/internal/grouper"
	"github.com/gopherjs/gopherjs/compiler/linkname"
	"github.com/gopherjs/gopherjs/compiler/sources"
	"github.com/gopherjs/gopherjs/internal/srctesting"
)

func TestOrder(t *testing.T) {
	fileA := `
		package foo

		var Avar = "a"

		type Atype struct{}

		func Afunc() int {
			var varA = 1
			var varB = 2
			return varA+varB
		}`

	fileB := `
		package foo

		var Bvar = "b"

		type Btype struct{}

		func Bfunc() int {
			var varA = 1
			var varB = 2
			return varA+varB
		}`

	files := []srctesting.Source{
		{Name: "fileA.go", Contents: []byte(fileA)},
		{Name: "fileB.go", Contents: []byte(fileB)},
	}

	compareOrder(t, files, false)
	compareOrder(t, files, true)
}

func TestDeclSelection_KeepUnusedExportedMethods(t *testing.T) {
	src := `
		package main
		type Foo struct {}
		func (f Foo) Bar() {
			println("bar")
		}
		func (f Foo) Baz() { // unused
			println("baz")
		}
		func main() {
			Foo{}.Bar()
		}`

	srcFiles := []srctesting.Source{{Name: `main.go`, Contents: []byte(src)}}
	sel := declSelection(t, srcFiles, nil)

	sel.IsAlive(`type:command-line-arguments.Foo`)
	sel.IsAlive(`func:command-line-arguments.Foo.Bar`)
	sel.IsAlive(`func:command-line-arguments.Foo.Baz`)
}

func TestDeclSelection_RemoveUnusedUnexportedMethods(t *testing.T) {
	src := `
		package main
		type Foo struct {}
		func (f Foo) Bar() {
			println("bar")
		}
		func (f Foo) baz() { // unused
			println("baz")
		}
		func main() {
			Foo{}.Bar()
		}`

	srcFiles := []srctesting.Source{{Name: `main.go`, Contents: []byte(src)}}
	sel := declSelection(t, srcFiles, nil)

	sel.IsAlive(`type:command-line-arguments.Foo`)
	sel.IsAlive(`func:command-line-arguments.Foo.Bar`)

	sel.IsDead(`func:command-line-arguments.Foo.baz`)
}

func TestDeclSelection_KeepUnusedUnexportedMethodForInterface(t *testing.T) {
	src := `
		package main
		type Foo struct {}
		func (f Foo) Bar() {
			println("foo")
		}
		func (f Foo) baz() {} // unused

		type Foo2 struct {}
		func (f Foo2) Bar() {
			println("foo2")
		}

		type IFoo interface {
			Bar()
			baz()
		}
		func main() {
			fs := []any{ Foo{}, Foo2{} }
			for _, f := range fs {
				if i, ok := f.(IFoo); ok {
					i.Bar()
				}
			}
		}`

	srcFiles := []srctesting.Source{{Name: `main.go`, Contents: []byte(src)}}
	sel := declSelection(t, srcFiles, nil)

	sel.IsAlive(`type:command-line-arguments.Foo`)
	sel.IsAlive(`func:command-line-arguments.Foo.Bar`)

	// `baz` signature metadata is used to check a type assertion against IFoo,
	// but the method itself is never called, so it can be removed.
	// The method is kept in Foo's MethodList for type checking.
	sel.IsDead(`func:command-line-arguments.Foo.baz`)
}

func TestDeclSelection_KeepUnexportedMethodUsedViaInterfaceLit(t *testing.T) {
	src := `
		package main
		type Foo struct {}
		func (f Foo) Bar() {
			println("foo")
		}
		func (f Foo) baz() {
			println("baz")
		}
		func main() {
			var f interface {
				Bar()
				baz()
			} = Foo{}
			f.baz()
		}`

	srcFiles := []srctesting.Source{{Name: `main.go`, Contents: []byte(src)}}
	sel := declSelection(t, srcFiles, nil)

	sel.IsAlive(`type:command-line-arguments.Foo`)
	sel.IsAlive(`func:command-line-arguments.Foo.Bar`)
	sel.IsAlive(`func:command-line-arguments.Foo.baz`)
}

func TestDeclSelection_KeepAliveUnexportedMethodsUsedInMethodExpressions(t *testing.T) {
	src := `
		package main
		type Foo struct {}
		func (f Foo) baz() {
			println("baz")
		}
		func main() {
			fb := Foo.baz
			fb(Foo{})
		}`

	srcFiles := []srctesting.Source{{Name: `main.go`, Contents: []byte(src)}}
	sel := declSelection(t, srcFiles, nil)

	sel.IsAlive(`type:command-line-arguments.Foo`)
	sel.IsAlive(`func:command-line-arguments.Foo.baz`)
}

func TestDeclSelection_RemoveUnusedFuncInstance(t *testing.T) {
	src := `
		package main
		func Sum[T int | float64](values ...T) T {
			var sum T
			for _, v := range values {
				sum += v
			}
			return sum
		}
		func Foo() { // unused
			println(Sum(1, 2, 3))
		}
		func main() {
			println(Sum(1.1, 2.2, 3.3))
		}`

	srcFiles := []srctesting.Source{{Name: `main.go`, Contents: []byte(src)}}
	sel := declSelection(t, srcFiles, nil)

	sel.IsAlive(`func:command-line-arguments.Sum<float64>`)
	sel.IsAlive(`anonType:command-line-arguments.sliceType$1`) // []float64

	sel.IsDead(`func:command-line-arguments.Foo`)
	sel.IsDead(`anonType:command-line-arguments.sliceType`) // []int
	sel.IsDead(`func:command-line-arguments.Sum<int>`)
}

func TestDeclSelection_RemoveUnusedStructTypeInstances(t *testing.T) {
	src := `
		package main
		type Foo[T any] struct { v T }
		func (f Foo[T]) Bar() {
			println(f.v)
		}
		
		var _ = Foo[float64]{v: 3.14} // unused

		func main() {
			Foo[int]{v: 7}.Bar()
		}`

	srcFiles := []srctesting.Source{{Name: `main.go`, Contents: []byte(src)}}
	sel := declSelection(t, srcFiles, nil)

	sel.IsAlive(`type:command-line-arguments.Foo<int>`)
	sel.IsAlive(`func:command-line-arguments.Foo.Bar<int>`)

	sel.IsDead(`type:command-line-arguments.Foo<float64>`)
	sel.IsDead(`func:command-line-arguments.Foo.Bar<float64>`)
}

func TestDeclSelection_RemoveUnusedInterfaceTypeInstances(t *testing.T) {
	src := `
		package main
		type Foo[T any] interface { Bar(v T) }

		type Baz int
		func (b Baz) Bar(v int) {
			println(v + int(b))
		}
		
		var F64 = FooBar[float64] // unused

		func FooBar[T any](f Foo[T], v T) {
			f.Bar(v)
		}

		func main() {
			FooBar[int](Baz(42), 12) // Baz implements Foo[int]
		}`

	srcFiles := []srctesting.Source{{Name: `main.go`, Contents: []byte(src)}}
	sel := declSelection(t, srcFiles, nil)

	sel.IsAlive(`type:command-line-arguments.Baz`)
	sel.IsAlive(`func:command-line-arguments.Baz.Bar`)
	sel.IsDead(`var:command-line-arguments.F64`)

	sel.IsAlive(`func:command-line-arguments.FooBar<int>`)
	// The Foo[int] instance is defined as a parameter in FooBar[int] that is alive.
	// However, Foo[int] isn't used directly in the code so it can be removed.
	// JS will simply duck-type the Baz object to Foo[int] without Foo[int] specifically defined.
	sel.IsDead(`type:command-line-arguments.Foo<int>`)

	sel.IsDead(`func:command-line-arguments.FooBar<float64>`)
	sel.IsDead(`type:command-line-arguments.Foo<float64>`)
}

func TestDeclSelection_RemoveUnusedMethodWithDifferentSignature(t *testing.T) {
	src := `
		package main
		type Foo struct{}
		func (f Foo) Bar() { println("Foo") }
		func (f Foo) baz(x int) { println(x) } // unused

		type Foo2 struct{}
		func (f Foo2) Bar() { println("Foo2") }
		func (f Foo2) baz(x string) { println(x) }
		
		func main() {
			f1 := Foo{}
			f1.Bar()

			f2 := Foo2{}
			f2.Bar()
			f2.baz("foo")
		}`

	srcFiles := []srctesting.Source{{Name: `main.go`, Contents: []byte(src)}}
	sel := declSelection(t, srcFiles, nil)

	sel.IsAlive(`type:command-line-arguments.Foo`)
	sel.IsAlive(`func:command-line-arguments.Foo.Bar`)
	sel.IsDead(`func:command-line-arguments.Foo.baz`)

	sel.IsAlive(`type:command-line-arguments.Foo2`)
	sel.IsAlive(`func:command-line-arguments.Foo2.Bar`)
	sel.IsAlive(`func:command-line-arguments.Foo2.baz`)
}

func TestDeclSelection_RemoveUnusedUnexportedMethodInstance(t *testing.T) {
	src := `
		package main
		type Foo[T any] struct{}
		func (f Foo[T]) Bar() { println("Foo") }
		func (f Foo[T]) baz(x T) { Baz[T]{v: x}.Bar() }

		type Baz[T any] struct{ v T }
		func (b Baz[T]) Bar() { println("Baz", b.v) }

		func main() {
			f1 := Foo[int]{}
			f1.Bar()
			f1.baz(7)

			f2 := Foo[uint]{} // Foo[uint].baz is unused
			f2.Bar()
		}`

	srcFiles := []srctesting.Source{{Name: `main.go`, Contents: []byte(src)}}
	sel := declSelection(t, srcFiles, nil)

	sel.IsAlive(`type:command-line-arguments.Foo<int>`)
	sel.IsAlive(`func:command-line-arguments.Foo.Bar<int>`)
	sel.IsAlive(`func:command-line-arguments.Foo.baz<int>`)
	sel.IsAlive(`type:command-line-arguments.Baz<int>`)
	sel.IsAlive(`func:command-line-arguments.Baz.Bar<int>`)

	sel.IsAlive(`type:command-line-arguments.Foo<uint>`)
	sel.IsAlive(`func:command-line-arguments.Foo.Bar<uint>`)

	// All three below are dead because Foo[uint].baz is unused.
	sel.IsDead(`func:command-line-arguments.Foo.baz<uint>`)
	sel.IsDead(`type:command-line-arguments.Baz<uint>`)
	sel.IsDead(`func:command-line-arguments.Baz.Bar<uint>`)
}

func TestDeclSelection_RemoveUnusedTypeConstraint(t *testing.T) {
	src := `
		package main
		type Foo interface{ int | string }

		type Bar[T Foo] struct{ v T }
		func (b Bar[T]) Baz() { println(b.v) }

		var ghost = Bar[int]{v: 7} // unused

		func main() {
			println("do nothing")
		}`

	srcFiles := []srctesting.Source{{Name: `main.go`, Contents: []byte(src)}}
	sel := declSelection(t, srcFiles, nil)

	sel.IsDead(`type:command-line-arguments.Foo`)
	sel.IsDead(`type:command-line-arguments.Bar<int>`)
	sel.IsDead(`func:command-line-arguments.Bar.Baz<int>`)
	sel.IsDead(`var:command-line-arguments.ghost`)
}

func TestDeclSelection_RemoveUnusedNestedTypesInFunction(t *testing.T) {
	src := `
		package main
		func Foo[T any](u T) any {
			type Bar struct { v T }
			return Bar{v: u}
		}
		func deadCode() {
			println(Foo[int](42))
		}
		func main() {
			println(Foo[string]("cat"))
		}`

	srcFiles := []srctesting.Source{{Name: `main.go`, Contents: []byte(src)}}
	sel := declSelection(t, srcFiles, nil)
	sel.IsAlive(`func:command-line-arguments.main`)

	sel.IsAlive(`funcVar:command-line-arguments.Foo`)
	sel.IsAlive(`func:command-line-arguments.Foo<string>`)
	sel.IsDead(`func:command-line-arguments.Foo<int>`)

	sel.IsAlive(`typeVar:command-line-arguments.Bar`)
	sel.IsAlive(`type:command-line-arguments.Bar<string;>`)
	sel.IsDead(`type:command-line-arguments.Bar<int;>`)

	sel.IsDead(`funcVar:command-line-arguments.deadCode`)
	sel.IsDead(`func:command-line-arguments.deadCode`)
}

func TestDeclSelection_RemoveUnusedNestedTypesInMethod(t *testing.T) {
	src := `
		package main
		type Baz[T any] struct{}
		func (b *Baz[T]) Foo(u T) any {
			type Bar struct { v T }
			return Bar{v: u}
		}
		func deadCode() {
			b := Baz[int]{}
			println(b.Foo(42))
		}
		func main() {
			b := Baz[string]{}
			println(b.Foo("cat"))
		}`

	srcFiles := []srctesting.Source{{Name: `main.go`, Contents: []byte(src)}}
	sel := declSelection(t, srcFiles, nil)
	sel.IsAlive(`func:command-line-arguments.main`)

	sel.IsAlive(`typeVar:command-line-arguments.Baz`)
	sel.IsDead(`type:command-line-arguments.Baz<int>`)
	sel.IsAlive(`type:command-line-arguments.Baz<string>`)

	sel.IsDead(`func:command-line-arguments.(*Baz).Foo<int>`)
	sel.IsAlive(`func:command-line-arguments.(*Baz).Foo<string>`)

	sel.IsAlive(`typeVar:command-line-arguments.Bar`)
	sel.IsDead(`type:command-line-arguments.Bar<int;>`)
	sel.IsAlive(`type:command-line-arguments.Bar<string;>`)

	sel.IsDead(`funcVar:command-line-arguments.deadCode`)
	sel.IsDead(`func:command-line-arguments.deadCode`)
}

func TestDeclSelection_RemoveAllUnusedNestedTypes(t *testing.T) {
	src := `
		package main
		func Foo[T any](u T) any {
			type Bar struct { v T }
			return Bar{v: u}
		}
		func deadCode() {
			println(Foo[int](42))
			println(Foo[string]("cat"))
		}
		func main() {}`

	srcFiles := []srctesting.Source{{Name: `main.go`, Contents: []byte(src)}}
	sel := declSelection(t, srcFiles, nil)
	sel.IsAlive(`func:command-line-arguments.main`)

	sel.IsDead(`funcVar:command-line-arguments.Foo`)
	sel.IsDead(`func:command-line-arguments.Foo<string>`)
	sel.IsDead(`func:command-line-arguments.Foo<int>`)

	sel.IsDead(`typeVar:command-line-arguments.Bar`)
	sel.IsDead(`type:command-line-arguments.Bar<string;>`)
	sel.IsDead(`type:command-line-arguments.Bar<int;>`)

	sel.IsDead(`funcVar:command-line-arguments.deadCode`)
	sel.IsDead(`func:command-line-arguments.deadCode`)
}

func TestDeclSelection_CompletelyRemoveNestedType(t *testing.T) {
	src := `
		package main
		func Foo[T any](u T) any {
			type Bar struct { v T }
			return Bar{v: u}
		}
		func deadCode() {
			println(Foo[int](42))
		}
		func main() {}`

	srcFiles := []srctesting.Source{{Name: `main.go`, Contents: []byte(src)}}
	sel := declSelection(t, srcFiles, nil)

	sel.IsAlive(`func:command-line-arguments.main`)

	sel.IsDead(`funcVar:command-line-arguments.Foo`)
	sel.IsDead(`func:command-line-arguments.Foo<int>`)

	sel.IsDead(`typeVar:command-line-arguments.Bar`)
	sel.IsDead(`type:command-line-arguments.Bar<int;>`)

	sel.IsDead(`funcVar:command-line-arguments.deadCode`)
	sel.IsDead(`func:command-line-arguments.deadCode`)
}

func TestDeclSelection_RemoveAnonNestedTypes(t *testing.T) {
	// Based on test/fixedbugs/issue53635.go
	// This checks that if an anon type (e.g. []T) is used in a function
	// that is not used, the type is removed, otherwise it is kept.

	src := `
		package main
		func Foo[T any](u T) any {
			return []T(nil)
		}
		func deadCode() {
			println(Foo[string]("cat"))
		}
		func main() {
			println(Foo[int](42))
		}`

	srcFiles := []srctesting.Source{{Name: `main.go`, Contents: []byte(src)}}
	sel := declSelection(t, srcFiles, nil)
	sel.IsDead(`anonType:command-line-arguments.sliceType`)    // []string
	sel.IsAlive(`anonType:command-line-arguments.sliceType$1`) // []int
}

func TestDeclSelection_NoNestAppliedToFuncCallInMethod(t *testing.T) {
	// Checks that a function call to a non-local function isn't
	// being labeled as a nested function call.
	src := `
		package main
		func foo(a any) {
			println(a)
		}
		type Bar[T any] struct { u T }
		func (b *Bar[T]) Baz() {
			foo(b.u)
		}
		func main() {
			b := &Bar[int]{u: 42}
			b.Baz()
		}`

	srcFiles := []srctesting.Source{{Name: `main.go`, Contents: []byte(src)}}
	sel := declSelection(t, srcFiles, nil)
	sel.IsAlive(`init:main`)

	sel.IsAlive(`typeVar:command-line-arguments.Bar`)
	sel.IsAlive(`type:command-line-arguments.Bar<int>`)
	sel.IsAlive(`func:command-line-arguments.(*Bar).Baz<int>`)

	sel.IsAlive(`func:command-line-arguments.foo`)
}

func TestLengthParenthesizingIssue841(t *testing.T) {
	// See issue https://github.com/gopherjs/gopherjs/issues/841
	//
	// Summary: Given `len(a+b)` where a and b are strings being concatenated
	// together, the result was `a + b.length` instead of `(a+b).length`.
	//
	// The fix was to check if the expression in `len` is a binary
	// expression or not. If it is, then the expression is parenthesized.
	// This will work for concatenations any combination of variables and
	// literals but won't pick up `len(Foo(a+b))` or `len(a[0:i+3])`.

	src := `
		package main

		func main() {
			a := "a"
			b := "b"
			ab := a + b
			if len(a+b) != len(ab) {
				panic("unreachable")
			}
		}`

	srcFiles := []srctesting.Source{{Name: `main.go`, Contents: []byte(src)}}
	root := srctesting.ParseSources(t, srcFiles, nil)
	archives := compileProject(t, root, nil, false)
	mainPkg := archives[root.PkgPath]

	badRegex := regexp.MustCompile(`a\s*\+\s*b\.length`)
	goodRegex := regexp.MustCompile(`\(a\s*\+\s*b\)\.length`)
	goodFound := false
	for i, decl := range mainPkg.Declarations {
		if badRegex.Match(decl.DeclCode) {
			t.Errorf("found length issue in decl #%d: %s", i, decl.FullName)
			t.Logf("decl code:\n%s", string(decl.DeclCode))
		}
		if goodRegex.Match(decl.DeclCode) {
			goodFound = true
		}
	}
	if !goodFound {
		t.Error("parenthesized length not found")
	}
}

func TestDeclNaming_Import(t *testing.T) {
	src1 := `
		package main
		
		import (
			newt "github.com/gopherjs/gopherjs/compiler/jorden"
			"github.com/gopherjs/gopherjs/compiler/burke"
			"github.com/gopherjs/gopherjs/compiler/hudson"
		)

		func main() {
			newt.Quote()
			burke.Quote()
			hudson.Quote()
		}`
	src2 := `package jorden
		func Quote() { println("They mostly come at night... mostly") }`
	src3 := `package burke
		func Quote() { println("Busy little creatures, huh?") }`
	src4 := `package hudson
		func Quote() { println("Game over, man! Game over!") }`

	root := srctesting.ParseSources(t,
		[]srctesting.Source{
			{Name: `main.go`, Contents: []byte(src1)},
		},
		[]srctesting.Source{
			{Name: `jorden/rebecca.go`, Contents: []byte(src2)},
			{Name: `burke/carter.go`, Contents: []byte(src3)},
			{Name: `hudson/william.go`, Contents: []byte(src4)},
		})

	archives := compileProject(t, root, nil, false)
	checkForDeclFullNames(t, archives,
		`import:github.com/gopherjs/gopherjs/compiler/burke`,
		`import:github.com/gopherjs/gopherjs/compiler/hudson`,
		`import:github.com/gopherjs/gopherjs/compiler/jorden`,
	)
}

func TestDeclNaming_FuncAndFuncVar(t *testing.T) {
	src := `
		package main
		
		func Avasarala(value int) { println("Chrisjen", value) }

		func Draper[T any](value T) { println("Bobbie", value) }

		type Nagata struct{ value int }
		func (n Nagata) Print() { println("Naomi", n.value) }

		type Burton[T any] struct{ value T }
		func (b Burton[T]) Print() { println("Amos", b.value) }

		func main() {
			Avasarala(10)
			Draper(11)
			Draper("Babs")
			Nagata{value: 12}.Print()
			Burton[int]{value: 13}.Print()
			Burton[string]{value: "Timothy"}.Print()
		}`

	srcFiles := []srctesting.Source{{Name: `main.go`, Contents: []byte(src)}}
	root := srctesting.ParseSources(t, srcFiles, nil)
	archives := compileProject(t, root, nil, false)
	checkForDeclFullNames(t, archives,
		`funcVar:command-line-arguments.Avasarala`,
		`func:command-line-arguments.Avasarala`,

		`funcVar:command-line-arguments.Draper`,
		`func:command-line-arguments.Draper<int>`,
		`func:command-line-arguments.Draper<string>`,

		`func:command-line-arguments.Nagata.Print`,

		`typeVar:command-line-arguments.Burton`,
		`type:command-line-arguments.Burton<int>`,
		`type:command-line-arguments.Burton<string>`,
		`func:command-line-arguments.Burton.Print<int>`,
		`func:command-line-arguments.Burton.Print<string>`,

		`funcVar:command-line-arguments.main`,
		`func:command-line-arguments.main`,
		`init:main`,
	)
}

func TestDeclNaming_InitsAndVars(t *testing.T) {
	src1 := `
		package main
		
		import (
			_ "github.com/gopherjs/gopherjs/compiler/spengler"
			_ "github.com/gopherjs/gopherjs/compiler/barrett"
			_ "github.com/gopherjs/gopherjs/compiler/tully"
		)

		var peck = "Walter"
		func init() { println(peck) }

		func main() {
			println("Janosz Poha")
		}`
	src2 := `package spengler
		func init() { println("Egon") }
		var egie = func() { println("Dirt Farmer") }
		func init() { egie() }`
	src3 := `package barrett
		func init() { println("Dana") }`
	src4 := `package barrett
		func init() { println("Zuul") }`
	src5 := `package barrett
		func init() { println("Gatekeeper") }`
	src6 := `package tully
		func init() { println("Louis") }`
	src7 := `package tully
		var keymaster = "Vinz Clortho"
		func init() { println(keymaster) }`

	root := srctesting.ParseSources(t,
		[]srctesting.Source{
			{Name: `main.go`, Contents: []byte(src1)},
		},
		[]srctesting.Source{
			{Name: `spengler/a.go`, Contents: []byte(src2)},
			{Name: `barrett/a.go`, Contents: []byte(src3)},
			{Name: `barrett/b.go`, Contents: []byte(src4)},
			{Name: `barrett/c.go`, Contents: []byte(src5)},
			{Name: `tully/a.go`, Contents: []byte(src6)},
			{Name: `tully/b.go`, Contents: []byte(src7)},
		})

	archives := compileProject(t, root, nil, false)
	checkForDeclFullNames(t, archives,
		// tully
		`var:github.com/gopherjs/gopherjs/compiler/tully.keymaster`,
		`funcVar:github.com/gopherjs/gopherjs/compiler/tully.init`,
		`funcVar:github.com/gopherjs/gopherjs/compiler/tully.init`,
		`func:github.com/gopherjs/gopherjs/compiler/tully.init`,
		`func:github.com/gopherjs/gopherjs/compiler/tully.init`,

		// spangler
		`var:github.com/gopherjs/gopherjs/compiler/spengler.egie`,
		`funcVar:github.com/gopherjs/gopherjs/compiler/spengler.init`,
		`funcVar:github.com/gopherjs/gopherjs/compiler/spengler.init`,
		`func:github.com/gopherjs/gopherjs/compiler/spengler.init`,
		`func:github.com/gopherjs/gopherjs/compiler/spengler.init`,

		// barrett
		`funcVar:github.com/gopherjs/gopherjs/compiler/barrett.init`,
		`funcVar:github.com/gopherjs/gopherjs/compiler/barrett.init`,
		`funcVar:github.com/gopherjs/gopherjs/compiler/barrett.init`,
		`func:github.com/gopherjs/gopherjs/compiler/barrett.init`,
		`func:github.com/gopherjs/gopherjs/compiler/barrett.init`,
		`func:github.com/gopherjs/gopherjs/compiler/barrett.init`,

		// main
		`var:command-line-arguments.peck`,
		`funcVar:command-line-arguments.init`,
		`func:command-line-arguments.init`,
		`funcVar:command-line-arguments.main`,
		`func:command-line-arguments.main`,
		`init:main`,
	)
}

func TestDeclNaming_VarsAndTypes(t *testing.T) {
	src := `
		package main
		
		var _, shawn, _ = func() (int, string, float64) {
			return 1, "Vizzini", 3.14
		}()

		var _ = func() string {
			return "Inigo Montoya"
		}()

		var fezzik = struct{ value int }{value: 7}
		var inigo = struct{ value string }{value: "Montoya"}

		type westley struct{ value string }

		func main() {}`

	srcFiles := []srctesting.Source{{Name: `main.go`, Contents: []byte(src)}}
	root := srctesting.ParseSources(t, srcFiles, nil)

	archives := compileProject(t, root, nil, false)
	checkForDeclFullNames(t, archives,
		`var:command-line-arguments.shawn`,
		`var:blank`,

		`var:command-line-arguments.fezzik`,
		`anonType:command-line-arguments.structType`,

		`var:command-line-arguments.inigo`,
		`anonType:command-line-arguments.structType$1`,

		`typeVar:command-line-arguments.westley`,
		`type:command-line-arguments.westley`,
	)
}

func Test_CrossPackageAnalysis(t *testing.T) {
	src1 := `
		package main
		import "github.com/gopherjs/gopherjs/compiler/stable"

		func main() {
			m := map[string]int{
				"one":   1,
				"two":   2,
				"three": 3,
			}
			stable.Print(m)
		}`
	src2 := `
		package collections
		import "github.com/gopherjs/gopherjs/compiler/cmp"
		
		func Keys[K cmp.Ordered, V any, M ~map[K]V](m M) []K {
			keys := make([]K, 0, len(m))
			for k := range m {
				keys = append(keys, k)
			}
			return keys
		}`
	src3 := `
		package collections
		import "github.com/gopherjs/gopherjs/compiler/cmp"
			
		func Values[K cmp.Ordered, V any, M ~map[K]V](m M) []V {
			values := make([]V, 0, len(m))
			for _, v := range m {
				values = append(values, v)
			}
			return values
		}`
	src4 := `
		package sorts
		import "github.com/gopherjs/gopherjs/compiler/cmp"
		
		func Pair[K cmp.Ordered, V any, SK ~[]K, SV ~[]V](k SK, v SV) {
			Bubble(len(k),
				func(i, j int) bool { return k[i] < k[j] },
				func(i, j int) { k[i], v[i], k[j], v[j] = k[j], v[j], k[i], v[i] })
		}

		func Bubble(length int, less func(i, j int) bool, swap func(i, j int)) {
			for i := 0; i < length; i++ {
				for j := i + 1; j < length; j++ {
					if less(j, i) {
						swap(i, j)
					}
				}
			}
		}`
	src5 := `
		package stable
		import (
			"github.com/gopherjs/gopherjs/compiler/collections"
			"github.com/gopherjs/gopherjs/compiler/sorts"
			"github.com/gopherjs/gopherjs/compiler/cmp"
		)

		func Print[K cmp.Ordered, V any, M ~map[K]V](m M) {
			keys := collections.Keys(m)
			values := collections.Values(m)
			sorts.Pair(keys, values)
			for i, k := range keys {
				println(i, k, values[i])
			}
		}`
	src6 := `
		package cmp
		type Ordered interface { ~int | ~uint | ~float64 | ~string }`

	root := srctesting.ParseSources(t,
		[]srctesting.Source{
			{Name: `main.go`, Contents: []byte(src1)},
		},
		[]srctesting.Source{
			{Name: `collections/keys.go`, Contents: []byte(src2)},
			{Name: `collections/values.go`, Contents: []byte(src3)},
			{Name: `sorts/sorts.go`, Contents: []byte(src4)},
			{Name: `stable/print.go`, Contents: []byte(src5)},
			{Name: `cmp/ordered.go`, Contents: []byte(src6)},
		})

	archives := compileProject(t, root, nil, false)
	checkForDeclFullNames(t, archives,
		// collections
		`funcVar:github.com/gopherjs/gopherjs/compiler/collections.Values`,
		`func:github.com/gopherjs/gopherjs/compiler/collections.Values<string, int, map[string]int>`,
		`funcVar:github.com/gopherjs/gopherjs/compiler/collections.Keys`,
		`func:github.com/gopherjs/gopherjs/compiler/collections.Keys<string, int, map[string]int>`,

		// sorts
		`funcVar:github.com/gopherjs/gopherjs/compiler/sorts.Pair`,
		`func:github.com/gopherjs/gopherjs/compiler/sorts.Pair<string, int, []string, []int>`,
		`funcVar:github.com/gopherjs/gopherjs/compiler/sorts.Bubble`,
		`func:github.com/gopherjs/gopherjs/compiler/sorts.Bubble`,

		// stable
		`funcVar:github.com/gopherjs/gopherjs/compiler/stable.Print`,
		`func:github.com/gopherjs/gopherjs/compiler/stable.Print<string, int, map[string]int>`,

		// main
		`init:main`,
	)
}

func Test_IndexedSelectors(t *testing.T) {
	src1 := `
		package main
		import "github.com/gopherjs/gopherjs/compiler/other"
		func main() {
			// Instance IndexExpr with a package SelectorExpr for a function call.
			other.PrintZero[int]()
			other.PrintZero[string]()

			// Instance IndexListExpr with a package SelectorExpr for a function call.
			other.PrintZeroZero[int, string]()

			// Index IndexExpr with a struct SelectorExpr for a function call.
			f := other.Foo{Ops: []func() {
				other.PrintZero[int],
				other.PrintZero[string],
				other.PrintZeroZero[int, string],
			}}
			f.Ops[0]()
			f.Ops[1]()

			// Index IndexExpr with a package/var SelectorExpr for a function call.
			other.Bar.Ops[0]()
			other.Baz[0]()

			// IndexExpr with a SelectorExpr for a cast
			_ = other.ZHandle[int](other.PrintZero[int])

			// IndexListExpr with a SelectorExpr for a cast
			_ = other.ZZHandle[int, string](other.PrintZeroZero[int, string])
		}`
	src2 := `
		package other
		func PrintZero[T any]() {
			var zero T
			println("Zero is ", zero)
		}
		func PrintZeroZero[T any, U any]() {
			PrintZero[T]()
			PrintZero[U]()
		}

		type ZHandle[T any] func()
		type ZZHandle[T any, U any] func()

		type Foo struct { Ops []func() }
		var Bar = Foo{Ops: []func() {
			PrintZero[int],
			PrintZero[string],
		}}
		var Baz = Bar.Ops`

	root := srctesting.ParseSources(t,
		[]srctesting.Source{
			{Name: `main.go`, Contents: []byte(src1)},
		},
		[]srctesting.Source{
			{Name: `other/other.go`, Contents: []byte(src2)},
		})

	archives := compileProject(t, root, nil, false)
	// We mostly are checking that the code was turned into decls correctly,
	// since the issue was that indexed selectors were not being handled correctly,
	// so if it didn't panic by this point, it should be fine.
	checkForDeclFullNames(t, archives,
		`func:command-line-arguments.main`,
		`type:github.com/gopherjs/gopherjs/compiler/other.ZHandle<int>`,
		`type:github.com/gopherjs/gopherjs/compiler/other.ZZHandle<int, string>`,
	)
}

func TestArchiveSelectionAfterSerialization(t *testing.T) {
	src := `
		package main
		type Foo interface{ int | string }

		type Bar[T Foo] struct{ v T }
		func (b Bar[T]) Baz() { println(b.v) }

		var ghost = Bar[int]{v: 7} // unused

		func main() {
			println("do nothing")
		}`
	srcFiles := []srctesting.Source{{Name: `main.go`, Contents: []byte(src)}}
	root := srctesting.ParseSources(t, srcFiles, nil)
	rootPath := root.PkgPath
	origArchives := compileProject(t, root, nil, false)
	readArchives := reloadCompiledProject(t, origArchives, rootPath)

	origJS := renderPackage(t, origArchives[rootPath], false)
	readJS := renderPackage(t, readArchives[rootPath], false)

	if diff := cmp.Diff(origJS, readJS); diff != "" {
		t.Errorf("the reloaded files produce different JS:\n%s", diff)
	}
}

func Test_OrderOfTypeInit_Simple(t *testing.T) {
	src1 := `
		package main
		import "github.com/gopherjs/gopherjs/compiler/collections"
		import "github.com/gopherjs/gopherjs/compiler/cat"
		import "github.com/gopherjs/gopherjs/compiler/box"

		func main() {
			s := collections.NewStack[box.Unboxer[cat.Cat]]()
			s.Push(box.Box(cat.Cat{Name: "Erwin"}))
			s.Push(box.Box(cat.Cat{Name: "Dirac"}))
			println(s.Pop().Unbox().Name)
		}`
	src2 := `
		package collections

		type Stack[T any] struct { values []T }

		func NewStack[T any]() *Stack[T] {
			return &Stack[T]{}
		}

		func (s *Stack[T]) Count() int {
			return len(s.values)
		}

		func (s *Stack[T]) Push(value T) {
			s.values = append(s.values, value)
		}

		func (s *Stack[T]) Pop() (value T) {
			if len(s.values) > 0 {
				maxIndex := len(s.values) - 1
				s.values, value = s.values[:maxIndex], s.values[maxIndex]
			}
			return
		}`
	src3 := `
		package cat

		type Cat struct { Name string }`
	src4 := `
		package box

		type Unboxer[T any] interface { Unbox() T }

		type boxImp[T any] struct { whatsInTheBox T }

		func Box[T any](value T) Unboxer[T] {
			return &boxImp[T]{whatsInTheBox: value}
		}

		func (b *boxImp[T]) Unbox() T { return b.whatsInTheBox }`

	sel := declSelection(t,
		[]srctesting.Source{
			{Name: `main.go`, Contents: []byte(src1)},
		},
		[]srctesting.Source{
			{Name: `collections/stack.go`, Contents: []byte(src2)},
			{Name: `cat/cat.go`, Contents: []byte(src3)},
			{Name: `box/box.go`, Contents: []byte(src4)},
		})

	// Group 0
	// (imports, typeVars, funcVars, and init:main are defaulted into group 0)
	// box
	sel.InGroup(0, `typeVar:github.com/gopherjs/gopherjs/compiler/box.Unboxer`) // type box.Unboxer[T]
	sel.InGroup(0, `typeVar:github.com/gopherjs/gopherjs/compiler/box.boxImp`)  // type box.boxImp[T]
	sel.InGroup(0, `funcVar:github.com/gopherjs/gopherjs/compiler/box.Box`)     // func box.Box[T]
	// cat
	sel.InGroup(0, `typeVar:github.com/gopherjs/gopherjs/compiler/cat.Cat`) // type cat.Cat
	sel.InGroup(0, `type:github.com/gopherjs/gopherjs/compiler/cat.Cat`)    // type cat.Cat
	// collections
	sel.InGroup(0, `typeVar:github.com/gopherjs/gopherjs/compiler/collections.Stack`)    // type collections.Stack[T]
	sel.InGroup(0, `funcVar:github.com/gopherjs/gopherjs/compiler/collections.NewStack`) // func collections.NewStack[T]
	// main
	sel.InGroup(0, `init:main`)
	sel.InGroup(0, `funcVar:command-line-arguments.main`)
	sel.InGroup(0, `func:command-line-arguments.main`)

	// Group 1
	// box
	sel.InGroup(1, `type:github.com/gopherjs/gopherjs/compiler/box.Unboxer<github.com/gopherjs/gopherjs/compiler/cat.Cat>`)         // box.Unboxer[cat.Cat]
	sel.InGroup(1, `type:github.com/gopherjs/gopherjs/compiler/box.boxImp<github.com/gopherjs/gopherjs/compiler/cat.Cat>`)          // box.boxImp[cat.Cat]
	sel.InGroup(1, `func:github.com/gopherjs/gopherjs/compiler/box.Box<github.com/gopherjs/gopherjs/compiler/cat.Cat>`)             // box.Box[cat.Cat]() box.Unboxer[cat.Cat]
	sel.InGroup(1, `func:github.com/gopherjs/gopherjs/compiler/box.(*boxImp).Unbox<github.com/gopherjs/gopherjs/compiler/cat.Cat>`) // box.boxImp[cat.Cat].Unbox
	sel.InGroup(1, `anonType:github.com/gopherjs/gopherjs/compiler/box.ptrType`)                                                    // *boxImp[cat.Cat]

	// Group 2
	// collections
	sel.InGroup(2, `anonType:github.com/gopherjs/gopherjs/compiler/collections.sliceType`)                                                                                           // []box.Unboxer[cat.Cat]
	sel.InGroup(2, `type:github.com/gopherjs/gopherjs/compiler/collections.Stack<github.com/gopherjs/gopherjs/compiler/box.Unboxer[github.com/gopherjs/gopherjs/compiler/cat.Cat]>`) // collections.Stack[box.Unboxer[cat.Cat]]
	sel.InGroup(2, `anonType:github.com/gopherjs/gopherjs/compiler/collections.ptrType`)                                                                                             // *collections.Stack[box.Unboxer[cat.Cat]]
	sel.InGroup(2, `func:github.com/gopherjs/gopherjs/compiler/collections.NewStack<github.com/gopherjs/gopherjs/compiler/box.Unboxer[github.com/gopherjs/gopherjs/compiler/cat.Cat]>`)
	sel.InGroup(2, `func:github.com/gopherjs/gopherjs/compiler/collections.(*Stack).Count<github.com/gopherjs/gopherjs/compiler/box.Unboxer[github.com/gopherjs/gopherjs/compiler/cat.Cat]>`)
	sel.InGroup(2, `func:github.com/gopherjs/gopherjs/compiler/collections.(*Stack).Push<github.com/gopherjs/gopherjs/compiler/box.Unboxer[github.com/gopherjs/gopherjs/compiler/cat.Cat]>`)
	sel.InGroup(2, `func:github.com/gopherjs/gopherjs/compiler/collections.(*Stack).Pop<github.com/gopherjs/gopherjs/compiler/box.Unboxer[github.com/gopherjs/gopherjs/compiler/cat.Cat]>`)
}

func Test_OrderOfTypeInit_PingPong(t *testing.T) {
	src1 := `
		package main
		import "github.com/gopherjs/gopherjs/compiler/collections"
		import "github.com/gopherjs/gopherjs/compiler/cat"

		func main() {
			s := collections.NewHashSet[cat.Cat[collections.BadHasher]]()
			s.Add(cat.Cat[collections.BadHasher]{Name: "Fluffy"})
			s.Add(cat.Cat[collections.BadHasher]{Name: "Mittens"})
			s.Add(cat.Cat[collections.BadHasher]{Name: "Whiskers"})
			println(s.Count(), "elements")
		}`
	src2 := `
		package collections

		// HashSet keeps a set of non-nil elements that have unique hashes.
		type HashSet[E Hashable] struct { data map[uint]E }

		func NewHashSet[E Hashable]() *HashSet[E] {
			return &HashSet[E]{ data: map[uint]E{} }
		}

		func (s *HashSet[E]) Add(e E) {
			s.data[e.Hash()] = e
		}

		func (s *HashSet[E]) Count() int {
			return len(s.data)
		}`
	src3 := `
		package collections

		type Hasher interface {
			Add(value uint)
			Sum() uint
		}
		
		type Hashable interface {
			Hash() uint
		}
		
		type BadHasher struct { value uint }

		func (h BadHasher) Add(value uint) { h.value += value }
		func (h BadHasher) Sum() uint      { return h.value }`
	src4 := `
		package cat
		import "github.com/gopherjs/gopherjs/compiler/collections"

		type Cat[H collections.Hasher] struct { Name string }

		func (c Cat[H]) Hash() uint {
			var h H
			for _, v := range []rune(c.Name) {
				h.Add(uint(v))
			}
			return h.Sum()
		}`

	sel := declSelection(t,
		[]srctesting.Source{
			{Name: `main.go`, Contents: []byte(src1)},
		},
		[]srctesting.Source{
			{Name: `collections/hashmap.go`, Contents: []byte(src2)},
			{Name: `collections/hashes.go`, Contents: []byte(src3)},
			{Name: `cat/cat.go`, Contents: []byte(src4)},
		})

	// Group 0
	// imports, funcVars, typevars, and init:main are in group 0 by default.
	sel.InGroup(0, `func:command-line-arguments.main`)
	sel.InGroup(0, `anonType:github.com/gopherjs/gopherjs/compiler/cat.sliceType`) // []rune
	sel.InGroup(0, `type:github.com/gopherjs/gopherjs/compiler/collections.BadHasher`)
	sel.InGroup(0, `func:github.com/gopherjs/gopherjs/compiler/collections.BadHasher.Add`)
	sel.InGroup(0, `func:github.com/gopherjs/gopherjs/compiler/collections.BadHasher.Sum`)

	// Group 1
	sel.InGroup(1, `type:github.com/gopherjs/gopherjs/compiler/cat.Cat<github.com/gopherjs/gopherjs/compiler/collections.BadHasher>`)
	sel.InGroup(1, `func:github.com/gopherjs/gopherjs/compiler/cat.Cat.Hash<github.com/gopherjs/gopherjs/compiler/collections.BadHasher>`)

	// Group 2
	sel.InGroup(2, `anonType:github.com/gopherjs/gopherjs/compiler/collections.mapType`) // map[uint]cat.Cat[collections.BadHasher]
	sel.InGroup(2, `type:github.com/gopherjs/gopherjs/compiler/collections.HashSet<github.com/gopherjs/gopherjs/compiler/cat.Cat[github.com/gopherjs/gopherjs/compiler/collections.BadHasher]>`)
	sel.InGroup(2, `func:github.com/gopherjs/gopherjs/compiler/collections.(*HashSet).Add<github.com/gopherjs/gopherjs/compiler/cat.Cat[github.com/gopherjs/gopherjs/compiler/collections.BadHasher]>`)
	sel.InGroup(2, `func:github.com/gopherjs/gopherjs/compiler/collections.(*HashSet).Count<github.com/gopherjs/gopherjs/compiler/cat.Cat[github.com/gopherjs/gopherjs/compiler/collections.BadHasher]>`)
	sel.InGroup(2, `anonType:github.com/gopherjs/gopherjs/compiler/collections.ptrType`) // *collections.HashSet[cat.Cat[collections.BadHasher]]
	sel.InGroup(2, `func:github.com/gopherjs/gopherjs/compiler/collections.NewHashSet<github.com/gopherjs/gopherjs/compiler/cat.Cat[github.com/gopherjs/gopherjs/compiler/collections.BadHasher]>`)
}

func Test_OrderOfTypeInit_HiddenParamMissingInterface(t *testing.T) {
	// If a type (typically an interface) is only used as a parameter or
	// a result in top-level functions, it will not be a DCE dependency
	// of any other declaration and therefore be considered dead.
	// Because of how JS works, this will not cause a problem when calling
	// the function.
	// If a function pointer to a top-level function (like done when using
	// reflections), the function pointer will define the parameters
	// and results, so that type will be alive.
	//
	// This test checks that the dead and missing type parameter will
	// not cause a problem with the type initialization ordering.
	src1 := `
		package main
		import "github.com/gopherjs/gopherjs/compiler/dragon"
		import "github.com/gopherjs/gopherjs/compiler/drawer"

		func main() {
			t := dragon.Trogdor[drawer.Cottages]{}
			t.Target = drawer.Cottages{}
			
			drawer.Draw(t)
		}`
	src2 := `
		package drawer
		import "github.com/gopherjs/gopherjs/compiler/dragon"

		type Cottages struct {}
		func (c Cottages) String() string {
			return "thatched-roof cottage"
		}

		func Draw[D dragon.Dragon](d D) {
			d.Burninate()
		}`
	src3 := `
		package dragon
		type Target interface{ String() string }
		type Dragon interface { Burninate() }

		type Trogdor[T Target] struct { Target T }
		func (t Trogdor[T]) Burninate() {
			println("burninating the " + t.Target.String())
		}`

	sel := declSelection(t,
		[]srctesting.Source{
			{Name: `main.go`, Contents: []byte(src1)},
		},
		[]srctesting.Source{
			{Name: `drawer/drawer.go`, Contents: []byte(src2)},
			{Name: `dragon/dragon.go`, Contents: []byte(src3)},
		})

	// command-line-arguments
	sel.IsAlive(`func:command-line-arguments.main`)
	sel.InGroup(0, `funcVar:command-line-arguments.main`)

	// drawer
	sel.IsAlive(`type:github.com/gopherjs/gopherjs/compiler/drawer.Cottages`)
	sel.InGroup(0, `type:github.com/gopherjs/gopherjs/compiler/drawer.Cottages`)

	sel.IsAlive(`funcVar:github.com/gopherjs/gopherjs/compiler/drawer.Draw`)
	sel.IsAlive(`func:github.com/gopherjs/gopherjs/compiler/drawer.Draw<github.com/gopherjs/gopherjs/compiler/dragon.Trogdor[github.com/gopherjs/gopherjs/compiler/drawer.Cottages]>`)
	sel.InGroup(2, `func:github.com/gopherjs/gopherjs/compiler/drawer.Draw<github.com/gopherjs/gopherjs/compiler/dragon.Trogdor[github.com/gopherjs/gopherjs/compiler/drawer.Cottages]>`)

	// dragon
	sel.IsDead(`type:github.com/gopherjs/gopherjs/compiler/dragon.Target`)
	sel.IsDead(`type:github.com/gopherjs/gopherjs/compiler/dragon.Dragon`)

	sel.IsAlive(`typeVar:github.com/gopherjs/gopherjs/compiler/dragon.Trogdor`)
	sel.IsAlive(`type:github.com/gopherjs/gopherjs/compiler/dragon.Trogdor<github.com/gopherjs/gopherjs/compiler/drawer.Cottages>`)
	sel.InGroup(1, `type:github.com/gopherjs/gopherjs/compiler/dragon.Trogdor<github.com/gopherjs/gopherjs/compiler/drawer.Cottages>`)
}

func TestNestedConcreteTypeInGenericFunc(t *testing.T) {
	// This is a test of a type defined inside a generic function
	// that uses the type parameter of the function as a field type.
	// The `T` type is unique for each instance of `F`.
	// The use of `A` as a field is do demonstrate the difference in the types
	// however even if T had no fields, the type would still be different.
	//
	// Change `print(F[?]())` to `fmt.Printf("%T\n", F[?]())` for
	// golang playground to print the type of T in the different F instances.
	// (I just didn't want this test to depend on `fmt` when it doesn't need to.)

	src := `
		package main
		func F[A any]() any {
			type T struct{
				a A
			}
			return T{}
		}
		func main() {
			type Int int
			print(F[int]())
			print(F[Int]())
		}
		`

	srcFiles := []srctesting.Source{{Name: `main.go`, Contents: []byte(src)}}
	root := srctesting.ParseSources(t, srcFiles, nil)
	archives := compileProject(t, root, nil, false)
	mainPkg := archives[root.PkgPath]
	insts := collectDeclInstances(t, mainPkg)

	exp := []string{
		`F[int]`,
		`F[main.Int]`,  // Go prints `F[main.Int·2]`
		`T[int;]`,      // `T` from `F[int]`      (Go prints `T[int]`)
		`T[main.Int;]`, // `T` from `F[main.Int]` (Go prints `T[main.Int·2]`)
	}
	if diff := cmp.Diff(exp, insts); len(diff) > 0 {
		t.Errorf("the instances of generics are different:\n%s", diff)
	}
}

func TestNestedGenericTypeInGenericFunc(t *testing.T) {
	// This is a subset of the type param nested test from the go repo.
	// See https://github.com/golang/go/blob/go1.19.13/test/typeparam/nested.go
	// The test is failing because nested types aren't being typed differently.
	// For example the type of `T[int]` below is different based on `F[X]`
	// instance for different `X` type parameters, hence Go prints the type as
	// `T[X;int]` instead of `T[int]`.

	src := `
		package main
		func F[A any]() any {
			type T[B any] struct{
				a A
				b B
			}
			return T[int]{}
		}
		func main() {
			type Int int
			print(F[int]())
			print(F[Int]())
		}
		`

	srcFiles := []srctesting.Source{{Name: `main.go`, Contents: []byte(src)}}
	root := srctesting.ParseSources(t, srcFiles, nil)
	archives := compileProject(t, root, nil, false)
	mainPkg := archives[root.PkgPath]
	insts := collectDeclInstances(t, mainPkg)

	exp := []string{
		`F[int]`,
		`F[main.Int]`,
		`T[int; int]`,
		`T[main.Int; int]`,
	}
	if diff := cmp.Diff(exp, insts); len(diff) > 0 {
		t.Errorf("the instances of generics are different:\n%s", diff)
	}
}

func TestNestedGenericTypeInGenericFuncWithSharedTArgs(t *testing.T) {
	src := `
		package main
		func F[A any]() any {
			type T[B any] struct {
				b B
			}
			return T[A]{}
		}
		func main() {
			type Int int
			print(F[int]())
			print(F[Int]())
		}`

	srcFiles := []srctesting.Source{{Name: `main.go`, Contents: []byte(src)}}
	root := srctesting.ParseSources(t, srcFiles, nil)
	archives := compileProject(t, root, nil, false)
	mainPkg := archives[root.PkgPath]
	insts := collectDeclInstances(t, mainPkg)

	exp := []string{
		`F[int]`,
		`F[main.Int]`,
		`T[int; int]`,
		`T[main.Int; main.Int]`,
		// Make sure that T[int;main.Int] and T[main.Int;int] aren't created.
	}
	if diff := cmp.Diff(exp, insts); len(diff) > 0 {
		t.Errorf("the instances of generics are different:\n%s", diff)
	}
}

func collectDeclInstances(t *testing.T, pkg *Archive) []string {
	t.Helper()

	// Regex to match strings like `Foo[42 /* bar */] =` and capture
	// the name (`Foo`), the index (`42`), and the instance type (`bar`).
	rex := regexp.MustCompile(`^\s*(\w+)\s*\[\s*(\d+)\s*\/\*(.+)\*\/\s*\]\s*\=`)

	// Collect all instances of generics (e.g. `Foo[bar] @ 2`) written to the decl code.
	insts := []string{}
	for _, decl := range pkg.Declarations {
		if match := rex.FindAllStringSubmatch(string(decl.DeclCode), 1); len(match) > 0 {
			instance := match[0][1] + `[` + strings.TrimSpace(match[0][3]) + `]`
			instance = strings.ReplaceAll(instance, `command-line-arguments`, pkg.Name)
			insts = append(insts, instance)
		}
	}
	sort.Strings(insts)
	return insts
}

func compareOrder(t *testing.T, sourceFiles []srctesting.Source, minify bool) {
	t.Helper()
	outputNormal := compile(t, sourceFiles, minify)

	// reverse the array
	for i, j := 0, len(sourceFiles)-1; i < j; i, j = i+1, j-1 {
		sourceFiles[i], sourceFiles[j] = sourceFiles[j], sourceFiles[i]
	}

	outputReversed := compile(t, sourceFiles, minify)

	if diff := cmp.Diff(outputNormal, outputReversed); diff != "" {
		t.Errorf("files in different order produce different JS:\n%s", diff)
	}
}

func compile(t *testing.T, sourceFiles []srctesting.Source, minify bool) string {
	t.Helper()
	rootPkg := srctesting.ParseSources(t, sourceFiles, nil)
	archives := compileProject(t, rootPkg, nil, minify)

	path := rootPkg.PkgPath
	a, ok := archives[path]
	if !ok {
		t.Fatalf(`root package not found in archives: %s`, path)
	}

	return renderPackage(t, a, minify)
}

// compileProject compiles the given root package and all packages imported by the root.
// This returns the compiled archives of all packages keyed by their import path.
func compileProject(t *testing.T, root *packages.Package, tContext *types.Context, minify bool) map[string]*Archive {
	t.Helper()
	pkgMap := map[string]*packages.Package{}
	packages.Visit([]*packages.Package{root}, nil, func(pkg *packages.Package) {
		pkgMap[pkg.PkgPath] = pkg
	})

	allSrcs := map[string]*sources.Sources{}
	for _, pkg := range pkgMap {
		srcs := &sources.Sources{
			ImportPath: pkg.PkgPath,
			Dir:        ``,
			Files:      pkg.Syntax,
			FileSet:    pkg.Fset,
		}
		allSrcs[pkg.PkgPath] = srcs
	}

	importer := func(path, srcDir string) (*sources.Sources, error) {
		srcs, ok := allSrcs[path]
		if !ok {
			t.Fatal(`package not found:`, path)
			return nil, nil
		}
		return srcs, nil
	}

	if tContext == nil {
		tContext = types.NewContext()
	}
	sortedSources := make([]*sources.Sources, 0, len(allSrcs))
	for _, srcs := range allSrcs {
		sortedSources = append(sortedSources, srcs)
	}
	sources.SortedSourcesSlice(sortedSources)
	PrepareAllSources(sortedSources, importer, tContext)

	archives := map[string]*Archive{}
	for _, srcs := range allSrcs {
		a, err := Compile(srcs, tContext, minify)
		if err != nil {
			t.Fatal(`failed to compile:`, err)
		}
		archives[srcs.ImportPath] = a
	}
	return archives
}

// newTime creates an arbitrary time.Time offset by the given number of seconds.
// This is useful for quickly creating times that are before or after another.
func newTime(seconds float64) time.Time {
	return time.Date(1969, 7, 20, 20, 17, 0, 0, time.UTC).
		Add(time.Duration(seconds * float64(time.Second)))
}

// reloadCompiledProject persists the given archives into memory then reloads
// them from memory to simulate a cache reload of a precompiled project.
func reloadCompiledProject(t *testing.T, archives map[string]*Archive, rootPkgPath string) map[string]*Archive {
	t.Helper()

	// TODO(grantnelson-wf): The tests using this function are out-of-date
	// since they are testing the old archive caching that has been disabled.
	// At some point, these tests should be updated to test any new caching
	// mechanism that is implemented or removed. As is this function is faking
	// the old recursive archive loading that is no longer used since it
	// doesn't allow cross package analysis for generings.

	buildTime := newTime(5.0)
	serialized := map[string][]byte{}
	for path, a := range archives {
		buf := &bytes.Buffer{}
		if err := WriteArchive(a, buildTime, buf); err != nil {
			t.Fatalf(`failed to write archive for %s: %v`, path, err)
		}
		serialized[path] = buf.Bytes()
	}

	srcModTime := newTime(0.0)
	reloadCache := map[string]*Archive{}
	type ImportContext struct {
		Packages      map[string]*types.Package
		ImportArchive func(path string) (*Archive, error)
	}
	var importContext *ImportContext
	importContext = &ImportContext{
		Packages: map[string]*types.Package{},
		ImportArchive: func(path string) (*Archive, error) {
			// find in local cache
			if a, ok := reloadCache[path]; ok {
				return a, nil
			}

			// deserialize archive
			buf, ok := serialized[path]
			if !ok {
				t.Fatalf(`archive not found for %s`, path)
			}
			a, _, err := ReadArchive(path, bytes.NewReader(buf), srcModTime, importContext.Packages)
			if err != nil {
				t.Fatalf(`failed to read archive for %s: %v`, path, err)
			}
			reloadCache[path] = a
			return a, nil
		},
	}

	_, err := importContext.ImportArchive(rootPkgPath)
	if err != nil {
		t.Fatal(`failed to reload archives:`, err)
	}
	return reloadCache
}

func renderPackage(t *testing.T, archive *Archive, minify bool) string {
	t.Helper()

	sel := &dce.Selector[*Decl]{}
	for _, d := range archive.Declarations {
		sel.Include(d, false)
	}
	selection := sel.AliveDecls()

	buf := &bytes.Buffer{}

	if err := WritePkgCode(archive, selection, linkname.GoLinknameSet{}, minify, &SourceMapFilter{Writer: buf}); err != nil {
		t.Fatal(err)
	}

	b := buf.String()
	if len(b) == 0 {
		t.Fatal(`render package had no output`)
	}
	return b
}

type selectionTester struct {
	t            *testing.T
	mainPkg      *Archive
	archives     map[string]*Archive
	packages     []*Archive
	dceSelection map[*Decl]struct{}
}

func declSelection(t *testing.T, sourceFiles []srctesting.Source, auxFiles []srctesting.Source) *selectionTester {
	t.Helper()
	root := srctesting.ParseSources(t, sourceFiles, auxFiles)
	tc := types.NewContext()
	archives := compileProject(t, root, tc, false)
	mainPkg := archives[root.PkgPath]

	paths := make([]string, 0, len(archives))
	for path := range archives {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	packages := make([]*Archive, 0, len(archives))
	for _, path := range paths {
		packages = append(packages, archives[path])
	}

	sel := &dce.Selector[*Decl]{}
	for _, pkg := range packages {
		for _, d := range pkg.Declarations {
			sel.Include(d, false)
		}
	}
	dceSelection := sel.AliveDecls()
	grouper.Group(dceSelection)

	return &selectionTester{
		t:            t,
		mainPkg:      mainPkg,
		archives:     archives,
		packages:     packages,
		dceSelection: dceSelection,
	}
}

func (st *selectionTester) PrintDeclStatus() {
	st.t.Helper()
	for _, pkg := range st.packages {
		st.t.Logf(`Package %s`, pkg.ImportPath)
		for _, decl := range pkg.Declarations {
			status := `[Dead] `
			if _, ok := st.dceSelection[decl]; ok {
				status = `[Alive]`
			}
			group := decl.Grouper().Group
			st.t.Logf(`  %s [%d] %q`, status, group, decl.FullName)
		}
	}
}

func (st *selectionTester) PrintOrderMermaid() {
	st.t.Helper()
	mermaid := grouper.ToMermaid(st.dceSelection, func(d *Decl) string {
		text := d.FullName
		text = strings.ReplaceAll(text, `github.com/gopherjs/gopherjs/compiler/`, ``)
		text = strings.ReplaceAll(text, `<`, `[`)
		text = strings.ReplaceAll(text, `>`, `]`)
		return text
	})
	st.t.Logf(`Mermaid:\n%s`, mermaid)
}

func (st *selectionTester) InGroup(group int, declFullName string) {
	st.t.Helper()
	decl := st.FindDecl(declFullName)
	got := decl.Grouper().Group
	if got != group {
		st.t.Errorf(`expected the decl %q to be in group %d, but it is in group %d`, declFullName, group, got)
	}
}

func (st *selectionTester) IsAlive(declFullName string) {
	st.t.Helper()
	decl := st.FindDecl(declFullName)
	if _, ok := st.dceSelection[decl]; !ok {
		st.t.Error(`expected the decl to be alive:`, declFullName)
	}
}

func (st *selectionTester) IsDead(declFullName string) {
	st.t.Helper()
	decl := st.FindDecl(declFullName)
	if _, ok := st.dceSelection[decl]; ok {
		st.t.Error(`expected the decl to be dead:`, declFullName)
	}
}

func (st *selectionTester) FindDecl(declFullName string) *Decl {
	st.t.Helper()
	var found *Decl
	for _, pkg := range st.packages {
		for _, d := range pkg.Declarations {
			if d.FullName == declFullName {
				if found != nil {
					st.t.Fatal(`multiple decls found with the name`, declFullName)
				}
				found = d
			}
		}
	}
	if found == nil {
		st.t.Fatal(`no decl found by the name`, declFullName)
	}
	return found
}

func checkForDeclFullNames(t *testing.T, archives map[string]*Archive, expectedFullNames ...string) {
	t.Helper()

	expected := map[string]int{}
	counts := map[string]int{}
	for _, name := range expectedFullNames {
		expected[name]++
		counts[name]++
	}
	for _, pkg := range archives {
		for _, decl := range pkg.Declarations {
			if found, has := expected[decl.FullName]; has {
				if found <= 0 {
					t.Errorf(`decl name existed more than %d time(s): %q`, counts[decl.FullName], decl.FullName)
				} else {
					expected[decl.FullName]--
				}
			}
		}
	}
	for imp, found := range expected {
		if found > 0 {
			t.Errorf(`missing %d decl name(s): %q`, found, imp)
		}
	}
	if t.Failed() {
		t.Log("Declarations:")
		for pkgName, pkg := range archives {
			t.Logf("\t%q", pkgName)
			for i, decl := range pkg.Declarations {
				t.Logf("\t\t%d:\t%q", i, decl.FullName)
			}
		}
	}
}
