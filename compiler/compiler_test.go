package compiler

import (
	"bytes"
	"go/types"
	"regexp"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/tools/go/packages"

	"github.com/gopherjs/gopherjs/compiler/internal/dce"
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
	archives := compileProject(t, root, false)
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

	archives := compileProject(t, root, false)
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
	archives := compileProject(t, root, false)
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

	archives := compileProject(t, root, false)
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

	archives := compileProject(t, root, false)
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

func compareOrder(t *testing.T, sourceFiles []srctesting.Source, minify bool) {
	t.Helper()
	outputNormal := compile(t, sourceFiles, minify)

	// reverse the array
	for i, j := 0, len(sourceFiles)-1; i < j; i, j = i+1, j-1 {
		sourceFiles[i], sourceFiles[j] = sourceFiles[j], sourceFiles[i]
	}

	outputReversed := compile(t, sourceFiles, minify)

	if diff := cmp.Diff(string(outputNormal), string(outputReversed)); diff != "" {
		t.Errorf("files in different order produce different JS:\n%s", diff)
	}
}

func compile(t *testing.T, sourceFiles []srctesting.Source, minify bool) []byte {
	t.Helper()
	rootPkg := srctesting.ParseSources(t, sourceFiles, nil)
	archives := compileProject(t, rootPkg, minify)

	path := rootPkg.PkgPath
	a, ok := archives[path]
	if !ok {
		t.Fatalf(`root package not found in archives: %s`, path)
	}

	b := renderPackage(t, a, minify)
	if len(b) == 0 {
		t.Fatal(`compile had no output`)
	}
	return b
}

// compileProject compiles the given root package and all packages imported by the root.
// This returns the compiled archives of all packages keyed by their import path.
func compileProject(t *testing.T, root *packages.Package, minify bool) map[string]*Archive {
	t.Helper()
	pkgMap := map[string]*packages.Package{}
	packages.Visit([]*packages.Package{root}, nil, func(pkg *packages.Package) {
		pkgMap[pkg.PkgPath] = pkg
	})

	archiveCache := map[string]*Archive{}
	var importContext *ImportContext
	importContext = &ImportContext{
		Packages: map[string]*types.Package{},
		Import: func(path string) (*Archive, error) {
			// find in local cache
			if a, ok := archiveCache[path]; ok {
				return a, nil
			}

			pkg, ok := pkgMap[path]
			if !ok {
				t.Fatal(`package not found:`, path)
			}
			importContext.Packages[path] = pkg.Types

			// compile package
			a, err := Compile(path, pkg.Syntax, pkg.Fset, importContext, minify)
			if err != nil {
				return nil, err
			}
			archiveCache[path] = a
			return a, nil
		},
	}

	_, err := importContext.Import(root.PkgPath)
	if err != nil {
		t.Fatal(`failed to compile:`, err)
	}
	return archiveCache
}

func renderPackage(t *testing.T, archive *Archive, minify bool) []byte {
	t.Helper()
	selection := make(map[*Decl]struct{})
	for _, d := range archive.Declarations {
		selection[d] = struct{}{}
	}

	buf := &bytes.Buffer{}

	if err := WritePkgCode(archive, selection, goLinknameSet{}, minify, &SourceMapFilter{Writer: buf}); err != nil {
		t.Fatal(err)
	}

	return buf.Bytes()
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
	archives := compileProject(t, root, false)
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
			if _, ok := st.dceSelection[decl]; ok {
				st.t.Logf(`  [Alive] %q`, decl.FullName)
			} else {
				st.t.Logf(`  [Dead]  %q`, decl.FullName)
			}
		}
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
