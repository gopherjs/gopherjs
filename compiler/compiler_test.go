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

	sel.DeclCode.IsAlive(`^\s*Foo = \$newType`)
	sel.DeclCode.IsAlive(`^\s*\$ptrType\(Foo\)\.prototype\.Bar`)
	sel.DeclCode.IsAlive(`^\s*\$ptrType\(Foo\)\.prototype\.Baz`)
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

	sel.DeclCode.IsAlive(`^\s*Foo = \$newType`)
	sel.DeclCode.IsAlive(`^\s*\$ptrType\(Foo\)\.prototype\.Bar`)

	sel.DeclCode.IsDead(`^\s*\$ptrType\(Foo\)\.prototype\.baz`)
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

	sel.DeclCode.IsAlive(`^\s*Foo = \$newType`)
	sel.DeclCode.IsAlive(`^\s*\$ptrType\(Foo\)\.prototype\.Bar`)

	// `baz` signature metadata is used to check a type assertion against IFoo,
	// but the method itself is never called, so it can be removed.
	sel.DeclCode.IsDead(`^\s*\$ptrType\(Foo\)\.prototype\.baz`)
	sel.MethodListCode.IsAlive(`^\s*Foo.methods = .* \{prop: "baz", name: "baz"`)
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

	sel.DeclCode.IsAlive(`^\s*Foo = \$newType`)
	sel.DeclCode.IsAlive(`^\s*\$ptrType\(Foo\)\.prototype\.Bar`)
	sel.DeclCode.IsAlive(`^\s*\$ptrType\(Foo\)\.prototype\.baz`)
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

	sel.DeclCode.IsAlive(`^\s*Foo = \$newType`)
	sel.DeclCode.IsAlive(`^\s*\$ptrType\(Foo\)\.prototype\.baz`)
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

	sel.DeclCode.IsAlive(`^\s*Sum\[\d+ /\* float64 \*/\]`)
	sel.DeclCode.IsAlive(`^\s*sliceType(\$\d+)? = \$sliceType\(\$Float64\)`)

	sel.DeclCode.IsDead(`^\s*Foo = function`)
	sel.DeclCode.IsDead(`^\s*sliceType(\$\d+)? = \$sliceType\(\$Int\)`)

	// TODO(grantnelson-wf): This should not be alive because it is not used.
	sel.DeclCode.IsAlive(`^\s*Sum\[\d+ /\* int \*/\]`)
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

	sel.DeclCode.IsAlive(`^\s*Foo\[\d+ /\* int \*/\] = \$newType`)
	sel.DeclCode.IsAlive(`^\s*\$ptrType\(Foo\[\d+ /\* int \*/\]\)\.prototype\.Bar`)

	// TODO(grantnelson-wf): This should not be alive because it is not used.
	sel.DeclCode.IsAlive(`^\s*Foo\[\d+ /\* float64 \*/\] = \$newType`)
	sel.DeclCode.IsAlive(`^\s*\$ptrType\(Foo\[\d+ /\* float64 \*/\]\)\.prototype\.Bar`)
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

	sel.DeclCode.IsAlive(`^\s*Baz = \$newType`)
	sel.DeclCode.IsAlive(`^\s*Baz\.prototype\.Bar`)
	sel.InitCode.IsDead(`\$pkg\.F64 = FooBar\[\d+ /\* float64 \*/\]`)

	sel.DeclCode.IsAlive(`^\s*FooBar\[\d+ /\* int \*/\]`)
	// TODO(grantnelson-wf): Below should be alive because it is an arg to FooBar[int].
	sel.DeclCode.IsDead(`^\s*Foo\[\d+ /\* int \*/\] = \$newType`)

	// TODO(grantnelson-wf): Below should be dead because it is only used by a dead init.
	sel.DeclCode.IsAlive(`^\s*FooBar\[\d+ /\* float64 \*/\]`)
	sel.DeclCode.IsDead(`^\s*Foo\[\d+ /\* float64 \*/\] = \$newType`)
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

	sel.DeclCode.IsAlive(`^\s*Foo = \$newType`)
	sel.DeclCode.IsAlive(`\s*\$ptrType\(Foo\)\.prototype\.Bar`)
	// TODO(grantnelson-wf): Below should be dead because it is not used even though
	// its name matches a used unexported method.
	sel.DeclCode.IsAlive(`\s*\$ptrType\(Foo\)\.prototype\.baz`)

	sel.DeclCode.IsAlive(`^\s*Foo2 = \$newType`)
	sel.DeclCode.IsAlive(`\s*\$ptrType\(Foo2\)\.prototype\.Bar`)
	sel.DeclCode.IsAlive(`\s*\$ptrType\(Foo2\)\.prototype\.baz`)
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

	sel.DeclCode.IsAlive(`^\s*Foo\[\d+ /\* int \*/\] = \$newType`)
	sel.DeclCode.IsAlive(`\s*\$ptrType\(Foo\[\d+ /\* int \*/\]\)\.prototype\.Bar`)
	sel.DeclCode.IsAlive(`\s*\$ptrType\(Foo\[\d+ /\* int \*/\]\)\.prototype\.baz`)
	sel.DeclCode.IsAlive(`^\s*Baz\[\d+ /\* int \*/\] = \$newType`)
	sel.DeclCode.IsAlive(`\s*\$ptrType\(Baz\[\d+ /\* int \*/\]\)\.prototype\.Bar`)

	sel.DeclCode.IsAlive(`^\s*Foo\[\d+ /\* uint \*/\] = \$newType`)
	sel.DeclCode.IsAlive(`\s*\$ptrType\(Foo\[\d+ /\* uint \*/\]\)\.prototype\.Bar`)
	// TODO(grantnelson-wf): All three below should be dead because Foo[uint].baz is unused.
	sel.DeclCode.IsAlive(`\s*\$ptrType\(Foo\[\d+ /\* uint \*/\]\)\.prototype\.baz`)
	sel.DeclCode.IsAlive(`^\s*Baz\[\d+ /\* uint \*/\] = \$newType`)
	sel.DeclCode.IsAlive(`\s*\$ptrType\(Baz\[\d+ /\* uint \*/\]\)\.prototype\.Bar`)
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

	sel.DeclCode.IsDead(`^\s*Foo = \$newType`)
	sel.DeclCode.IsDead(`^\s*Bar\[\d+ /\* int \*/\] = \$newType`)
	sel.DeclCode.IsDead(`^\s*\$ptrType\(Bar\[\d+ /\* int \*/\]\)\.prototype\.Baz`)
	sel.InitCode.IsDead(`ghost = new Bar\[\d+ /\* int \*/\]\.ptr\(7\)`)
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

	DeclCode       *selectionCodeTester
	InitCode       *selectionCodeTester
	MethodListCode *selectionCodeTester
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
		DeclCode: &selectionCodeTester{
			t:            t,
			packages:     packages,
			dceSelection: dceSelection,
			codeName:     `DeclCode`,
			getCode:      func(d *Decl) []byte { return d.DeclCode },
		},
		InitCode: &selectionCodeTester{
			t:            t,
			packages:     packages,
			dceSelection: dceSelection,
			codeName:     `InitCode`,
			getCode:      func(d *Decl) []byte { return d.InitCode },
		},
		MethodListCode: &selectionCodeTester{
			t:            t,
			packages:     packages,
			dceSelection: dceSelection,
			codeName:     `MethodListCode`,
			getCode:      func(d *Decl) []byte { return d.MethodListCode },
		},
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
			if len(decl.DeclCode) > 0 {
				st.t.Logf(`     DeclCode: %q`, string(decl.DeclCode))
			}
			if len(decl.InitCode) > 0 {
				st.t.Logf(`     InitCode: %q`, string(decl.InitCode))
			}
			if len(decl.MethodListCode) > 0 {
				st.t.Logf(`     MethodListCode: %q`, string(decl.MethodListCode))
			}
			if len(decl.TypeInitCode) > 0 {
				st.t.Logf(`     TypeInitCode: %q`, string(decl.TypeInitCode))
			}
			if len(decl.Vars) > 0 {
				st.t.Logf(`     Vars: %v`, decl.Vars)
			}
		}
	}
}

type selectionCodeTester struct {
	t            *testing.T
	packages     []*Archive
	dceSelection map[*Decl]struct{}
	codeName     string
	getCode      func(*Decl) []byte
}

func (ct *selectionCodeTester) IsAlive(pattern string) {
	ct.t.Helper()
	decl := ct.FindDeclMatch(pattern)
	if _, ok := ct.dceSelection[decl]; !ok {
		ct.t.Error(`expected the`, ct.codeName, `code to be alive:`, pattern)
	}
}

func (ct *selectionCodeTester) IsDead(pattern string) {
	ct.t.Helper()
	decl := ct.FindDeclMatch(pattern)
	if _, ok := ct.dceSelection[decl]; ok {
		ct.t.Error(`expected the`, ct.codeName, `code to be dead:`, pattern)
	}
}

func (ct *selectionCodeTester) FindDeclMatch(pattern string) *Decl {
	ct.t.Helper()
	regex := regexp.MustCompile(pattern)
	var found *Decl
	for _, pkg := range ct.packages {
		for _, d := range pkg.Declarations {
			if regex.Match(ct.getCode(d)) {
				if found != nil {
					ct.t.Fatal(`multiple`, ct.codeName, `found containing pattern:`, pattern)
				}
				found = d
			}
		}
	}
	if found == nil {
		ct.t.Fatal(ct.codeName, `not found with pattern:`, pattern)
	}
	return found
}
