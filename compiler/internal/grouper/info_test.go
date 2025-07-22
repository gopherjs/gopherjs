package grouper

import (
	"go/ast"
	"go/types"
	"path"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/gopherjs/gopherjs/compiler/internal/typeparams"
	"github.com/gopherjs/gopherjs/compiler/typesutil"
	"github.com/gopherjs/gopherjs/internal/srctesting"
)

func TestInstanceDecomposition(t *testing.T) {
	type testData struct {
		name     string
		context  *types.Context
		instance typeparams.Instance
		usePkg   bool
		expName  *types.Named
		expDeps  map[*types.Named]struct{}
	}

	tests := []testData{
		func() testData {
			tg := readTypes(t, `type Foo[T, U, V any] struct {}`)
			return testData{
				name:    `do not depend on basic types`,
				context: tg.tf.Context,
				instance: typeparams.Instance{
					Object: tg.Object(`Foo`),
					TArgs:  tg.TypeList(`int`, `string`, `bool`),
				},
				expName: tg.Named(`Foo[int, string, bool]`),
				expDeps: nil,
			}
		}(),
		func() testData {
			tg := readTypes(t, `type Foo[T, U any] struct {}`)
			return testData{
				name:    `do not depend on empty any or error`,
				context: tg.tf.Context,
				instance: typeparams.Instance{
					Object: tg.Object(`Foo`),
					TArgs:  tg.TypeList(`any`, `error`),
				},
				expName: tg.Named(`Foo[any, error]`),
				expDeps: nil,
			}
		}(),
		func() testData {
			tg := readTypes(t, `
				type Foo[T, U any] struct {}
				type Baz[V any] struct {}`)
			return testData{
				name:    `depend on type parameters`,
				context: tg.tf.Context,
				instance: typeparams.Instance{
					Object: tg.Object(`Foo`),
					TArgs:  tg.TypeList(`Baz[any]`, `Foo[int, bool]`),
				},
				expName: tg.Named(`Foo[Baz[any], Foo[int, bool]]`),
				expDeps: tg.NamedSet(`Baz[any]`, `Foo[int, bool]`),
			}
		}(),
		func() testData {
			tg := readTypes(t, `
				type Foo struct {}
				var f *Foo`)
			return testData{
				name:    `depend on pointer element`,
				context: tg.tf.Context,
				instance: typeparams.Instance{
					Object: tg.Object(`f`),
				},
				expName: nil, // `*Foo` is not named so it can't be depended on by name
				expDeps: tg.NamedSet(`Foo`),
			}
		}(),
		func() testData {
			tg := readTypes(t, `
				type Foo struct {}
				var s []Foo`)
			return testData{
				name:    `depend on slice element`,
				context: tg.tf.Context,
				instance: typeparams.Instance{
					Object: tg.Object(`s`),
				},
				expName: nil, // `[]Foo` is not named
				expDeps: tg.NamedSet(`Foo`),
			}
		}(),
		func() testData {
			tg := readTypes(t, `
				type Foo struct {}
				var c chan Foo`)
			return testData{
				name:    `depend on chan element`,
				context: tg.tf.Context,
				instance: typeparams.Instance{
					Object: tg.Object(`c`),
				},
				expName: nil, // `chan Foo` is not named
				expDeps: tg.NamedSet(`Foo`),
			}
		}(),
		func() testData {
			tg := readTypes(t, `
				type Foo struct {}
				type Bar struct {}
				var m map[Bar]Foo`)
			return testData{
				name:    `depend on map key and element`,
				context: tg.tf.Context,
				instance: typeparams.Instance{
					Object: tg.Object(`m`),
				},
				expName: nil, // `map[Bar]Foo` is not named
				expDeps: tg.NamedSet(`Bar`, `Foo`),
			}
		}(),
		func() testData {
			tg := readTypes(t, `
				type Foo struct { X Bar[Baz] }
				type Bar[T any] struct {}
				type Baz struct {}`)
			return testData{
				name:    `do not depend on fields`,
				context: tg.tf.Context,
				instance: typeparams.Instance{
					Object: tg.Object(`Foo`),
				},
				expName: tg.Named(`Foo`),
				expDeps: nil,
			}
		}(),
		func() testData {
			tg := readTypes(t, `
				type Foo func(p *Baz) []*Taz
				type Baz struct {}
				type Taz struct {}`)
			return testData{
				name:    `depend on parameter and result types`,
				context: tg.tf.Context,
				instance: typeparams.Instance{
					Object: tg.Object(`Foo`),
				},
				expName: tg.Named(`Foo`),
				expDeps: tg.NamedSet(`Baz`, `Taz`),
			}
		}(),
		func() testData {
			tg := readTypes(t, `
					type Foo[T any] func(x []*Foo[T]) map[string]*Foo[T]
					type Baz struct {}`)
			return testData{
				name:    `depend on resolved parameters and results`,
				context: tg.tf.Context,
				instance: typeparams.Instance{
					Object: tg.Object(`Foo`),
					TArgs:  tg.TypeList(`Baz`),
				},
				expName: tg.Named(`Foo[Baz]`),
				expDeps: tg.NamedSet(`Baz`, `Foo[Baz]`),
			}
		}(),
		func() testData {
			tg := readTypes(t, `
					type Foo[T any] struct {}
					type Bar struct {}
					var Baz = Foo[Bar]{}`)
			return testData{
				name:    `variables depend on the named in their type`,
				context: tg.tf.Context,
				instance: typeparams.Instance{
					Object: tg.Object(`Baz`),
				},
				expName: tg.Named(`Foo[Bar]`),
				expDeps: tg.NamedSet(`Bar`),
			}
		}(),
		func() testData {
			tg := readTypes(t, `
					type Foo struct{}
					type Bar []*Foo
					type Baz Bar`)
			return testData{
				name:    `dependency on underlying types for aliased types`,
				context: tg.tf.Context,
				instance: typeparams.Instance{
					Object: tg.Object(`Baz`),
				},
				expName: tg.Named(`Baz`),
				expDeps: tg.NamedSet(`Foo`),
			}
		}(),
		func() testData {
			tg := readTypes(t, `
					func Foo[T any]() any {
						type Bar struct{ x T}
						return Bar{}
					}
					type Baz struct{}`)
			return testData{
				name:    `depend on implicit nesting type arguments`,
				context: tg.tf.Context,
				instance: typeparams.Instance{
					Object: tg.Object(`Foo.Bar`),
					TNest:  tg.TypeList(`Baz`),
				},
				expName: tg.Object(`Foo.Bar`).Type().(*types.Named),
				expDeps: tg.NamedSet(`Baz`),
			}
		}(),
		func() testData {
			tg := readPackages(t, []srctesting.Source{{
				Name: `test.go`,
				Contents: []byte(
					`package testcase
						import "other"
						type Bar map[*other.Foo]int
						type Baz []Bar`),
			}, {
				Name: `other/other.go`,
				Contents: []byte(
					`package other
						type Foo struct{}`),
			}})
			return testData{
				name:    `depend on types from other packages`,
				context: tg.tf.Context,
				usePkg:  true,
				instance: typeparams.Instance{
					Object: tg.Object(`Baz`),
				},
				expName: tg.Named(`Baz`),
				// skip Bar since it is in the same package, instead dig into
				// Bar to find dependencies from other packages.
				expDeps: tg.NamedSet(`other.Foo`),
			}
		}(),
		func() testData {
			tg := readPackages(t, []srctesting.Source{{
				Name: `test.go`,
				Contents: []byte(
					`package testcase
						import "other"
						var Foo *other.Bar`),
			}, {
				Name: `other/other.go`,
				Contents: []byte(
					`package other
						type Bar struct{}`),
			}})
			return testData{
				name:    `depend on types behind pointers`,
				context: tg.tf.Context,
				usePkg:  true,
				instance: typeparams.Instance{
					Object: tg.Object(`Foo`),
				},
				expName: nil,
				expDeps: tg.NamedSet(`other.Bar`),
			}
		}(),
		// TODO(grantnelson-wf): Add tests for:
		// - concrete interfaces
		// - generic interfaces
		// - interfaces with embedded types
		// - `type x[T any] int` with methods that use `T`
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			info := &Info{}
			// Instead of calling SetInstance, we manually set the type and
			// dependencies so that we can use nil for the package, which will
			// disable the same package checks and make it easier to test.
			var pkg *types.Package
			if test.usePkg {
				pkg = test.instance.Object.Pkg()
			}
			info.setType(test.context, test.instance)
			info.addAllDeps(test.context, test.instance, pkg)

			if info.name != test.expName {
				t.Errorf("expected type %v, got %v", test.expName, info.name)
			}
			if diff := cmp.Diff(test.expDeps, info.dep); diff != "" {
				t.Errorf("unexpected dependencies (-want +got):\n%s", diff)
			}
		})
	}
}

const defaultImportPath = `pkg/test`

type typeGetter struct {
	tf    *srctesting.Fixture
	cache map[string]types.Type
}

func readTypes(t *testing.T, src string) typeGetter {
	t.Helper()
	return readPackages(t, []srctesting.Source{
		{Name: `test.go`, Contents: []byte("package testcase\n" + src)},
	})
}

func readPackages(t *testing.T, sources []srctesting.Source) typeGetter {
	t.Helper()
	tf := srctesting.New(t)

	// Parse all source files.
	pkgFiles := map[string][]*ast.File{}
	for _, s := range sources {
		importPath, filename := path.Split(s.Name)
		if len(importPath) == 0 {
			importPath = defaultImportPath
		}
		importPath = strings.TrimSuffix(importPath, `/`)
		file := tf.Parse(filename, string(s.Contents))
		pkgFiles[importPath] = append(pkgFiles[importPath], file)
	}

	// Create packages from parsed files.
	done := false
	for !done {
		done = true
		for importPath, files := range pkgFiles {
			if importsReady(tf, files) {
				tf.Check(importPath, files...)
				delete(pkgFiles, importPath)
				done = false
			}
		}
		if done && len(pkgFiles) > 0 {
			t.Fatalf("Failed to resolve imports for packages: %v", pkgFiles)
		}
	}

	return typeGetter{
		tf:    tf,
		cache: make(map[string]types.Type),
	}
}

func importsReady(tf *srctesting.Fixture, files []*ast.File) bool {
	tf.T.Helper()
	for _, file := range files {
		for _, imp := range file.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			if _, exists := tf.Packages[path]; !exists {
				return false
			}
		}
	}
	return true
}

func (tg typeGetter) Object(name string) types.Object {
	tg.tf.T.Helper()
	importPath := defaultImportPath
	if path, remainder, found := strings.Cut(name, `.`); found {
		if _, has := tg.tf.Packages[path]; has {
			importPath, name = path, remainder
		}
	}
	pkg := tg.tf.Packages[importPath]
	if pkg == nil {
		tg.tf.T.Fatalf(`missing package %q in fixture`, importPath)
	}
	return srctesting.LookupObj(pkg, name)
}

func (tg typeGetter) Type(expr string) types.Type {
	tg.tf.T.Helper()
	return tg.TypeList(expr)[0]
}

func (tg typeGetter) TypeList(exprs ...string) typesutil.TypeList {
	tg.tf.T.Helper()

	// Check the cache and determine which expressions need to be type-checked.
	result := make([]types.Type, len(exprs))
	missing := []int{}
	for i, e := range exprs {
		if typ, ok := tg.cache[e]; ok {
			result[i] = typ
		} else {
			missing = append(missing, i)
		}
	}
	if len(missing) == 0 {
		return result
	}

	// Create a faux file to type-check the expressions with.
	// The expressions are checked from the perspective of the root package.
	pkg := tg.tf.Packages[defaultImportPath]
	imports := []string{}
	for paths := range tg.tf.Packages {
		if paths != defaultImportPath {
			imports = append(imports, paths)
		}
	}
	faux := []string{`package ` + pkg.Name()}
	for _, path := range imports {
		faux = append(faux, `import "`+path+`"`)
	}
	for _, i := range missing {
		faux = append(faux, `var _ `+exprs[i])
	}
	f := tg.tf.Parse(`faux`, strings.Join(faux, "\n"))
	config := &types.Config{
		Context:                  tg.tf.Context,
		Sizes:                    &types.StdSizes{WordSize: 4, MaxAlign: 8},
		Importer:                 tg.tf,
		DisableUnusedImportCheck: true,
	}
	ck := types.NewChecker(config, tg.tf.FileSet, pkg, tg.tf.Info)
	if err := ck.Files([]*ast.File{f}); err != nil {
		tg.tf.T.Fatalf(`failed to type check expressions %v: %v`, missing, err)
	}

	// Extract the types from the type-checked file to fill in the result.
	index := 0
	ast.Inspect(f, func(node ast.Node) bool {
		if spec, ok := node.(*ast.ValueSpec); ok {
			typ := tg.tf.Info.Types[spec.Type].Type
			i := missing[index]
			tg.cache[exprs[i]] = typ
			result[i] = typ
			index++
			return false
		}
		return true
	})
	return result
}

func (tg typeGetter) Named(expr string) *types.Named {
	tg.tf.T.Helper()
	return tg.Type(expr).(*types.Named)
}

func (tg typeGetter) NamedSet(exprs ...string) map[*types.Named]struct{} {
	tg.tf.T.Helper()
	tl := tg.TypeList(exprs...)
	result := make(map[*types.Named]struct{}, len(exprs))
	for _, t := range tl {
		result[t.(*types.Named)] = struct{}{}
	}
	return result
}
