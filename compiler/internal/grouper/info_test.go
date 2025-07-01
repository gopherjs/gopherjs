package grouper

import (
	"go/ast"
	"go/types"
	"strings"
	"testing"

	"github.com/gopherjs/gopherjs/compiler/internal/typeparams"
	"github.com/gopherjs/gopherjs/compiler/typesutil"
	"github.com/gopherjs/gopherjs/internal/srctesting"
)

func TestInstanceDecomposition(t *testing.T) {
	type testData struct {
		name     string
		context  *types.Context
		instance typeparams.Instance
		expTyp   types.Type
		expDeps  []types.Type
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
				expTyp:  tg.Type(`Foo[int, string, bool]`),
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
				expTyp:  tg.Type(`Foo[any, error]`),
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
				expTyp:  tg.Type(`Foo[Baz[any], Foo[int, bool]]`),
				expDeps: tg.TypeList(`Baz[any]`, `Foo[int, bool]`),
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
				expTyp:  tg.Type(`*Foo`),
				expDeps: tg.TypeList(`Foo`),
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
				expTyp:  tg.Type(`[]Foo`),
				expDeps: tg.TypeList(`Foo`),
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
				expTyp:  tg.Type(`chan Foo`),
				expDeps: tg.TypeList(`Foo`),
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
				expTyp:  tg.Type(`map[Bar]Foo`),
				expDeps: tg.TypeList(`Bar`, `Foo`),
			}
		}(),
		func() testData {
			tg := readTypes(t, `
				type Foo struct { X Bar }
				type Bar struct {}`)
			return testData{
				name:    `do not need to depend on fields`,
				context: tg.tf.Context,
				instance: typeparams.Instance{
					Object: tg.Object(`Foo`),
				},
				expTyp:  tg.Type(`Foo`),
				expDeps: nil,
			}
		}(),
		func() testData {
			tg := readTypes(t, `
				type Foo struct {}
				func (f Foo) Bar(x int, y int) {}`)
			return testData{
				name:    `depend on receiver type and do not type methods`,
				context: tg.tf.Context,
				instance: typeparams.Instance{
					Object: tg.Object(`Foo.Bar`),
				},
				expTyp:  nil,
				expDeps: tg.TypeList(`Foo`),
			}
		}(),
		func() testData {
			tg := readTypes(t, `
				type Foo struct {}
				func (f *Foo) Bar(x int, y int) {}`)
			return testData{
				name:    `depend on receiver type without the pointer`,
				context: tg.tf.Context,
				instance: typeparams.Instance{
					Object: tg.Object(`Foo.Bar`),
				},
				expTyp:  nil,
				expDeps: tg.TypeList(`Foo`),
			}
		}(),
		func() testData {
			tg := readTypes(t, `
				type Foo[T any] struct {}
				func Bar[T any](x *Foo[T]) []*Foo[T] { return nil }
				type Baz struct {}`)
			return testData{
				name:    `depend on type arguments but not parameters nor results`,
				context: tg.tf.Context,
				instance: typeparams.Instance{
					Object: tg.Object(`Bar`),
					TArgs:  tg.TypeList(`Baz`),
				},
				expTyp:  nil,
				expDeps: tg.TypeList(`Baz`),
			}
		}(),
		func() testData {
			tg := readTypes(t, `
				type Foo[T any] struct {}
				type Bar struct {}
				var Baz = Foo[Bar]{}`)
			return testData{
				name:    `variables get typed and depend on their type parts`,
				context: tg.tf.Context,
				instance: typeparams.Instance{
					Object: tg.Object(`Baz`),
				},
				expTyp:  tg.Type(`Foo[Bar]`),
				expDeps: tg.TypeList(`Bar`),
			}
		}(),
		func() testData {
			tg := readTypes(t, `
				var Foo []struct{}`)
			return testData{
				name:    `do not depend on empty structs`,
				context: tg.tf.Context,
				instance: typeparams.Instance{
					Object: tg.Object(`Foo`),
				},
				expTyp:  tg.Type(`[]struct{}`),
				expDeps: nil,
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
				expTyp:  tg.Object(`Foo.Bar`).Type(),
				expDeps: tg.TypeList(`Baz`),
			}
		}(),
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			info := &Info{}
			info.SetInstance(test.context, test.instance)
			if !types.Identical(info.typ, test.expTyp) {
				t.Errorf("expected type %v, got %v", test.expTyp, info.typ)
			}

			if len(info.dep) != len(test.expDeps) {
				t.Errorf("expected %d dependencies, got %d", len(test.expDeps), len(info.dep))
				t.Log("\texpected:    ", test.expDeps)
				t.Log("\tdependencies:", info.dep)
			} else {
				dups := map[types.Type]bool{}
				failed := false
				for _, dep := range test.expDeps {
					if dups[dep] {
						t.Fatalf("duplicate dependency %v found in expected dependencies", dep)
					}
					dups[dep] = true
					if _, ok := info.dep[dep]; !ok {
						t.Errorf("expected dependency %v not found in %v", dep, info.dep)
						failed = true
					}
				}
				if failed {
					t.Log("\texpected:    ", test.expDeps)
					t.Log("\tdependencies:", info.dep)
				}
			}
		})
	}
}

type typeGetter struct {
	tf    *srctesting.Fixture
	cache map[string]types.Type
}

func readTypes(t *testing.T, src string) typeGetter {
	t.Helper()
	tf := srctesting.New(t)
	tf.Check(`pkg/test`, tf.Parse(`test.go`, "package testcase\n"+src))
	return typeGetter{
		tf:    tf,
		cache: make(map[string]types.Type),
	}
}

func (tg typeGetter) Object(name string) types.Object {
	tg.tf.T.Helper()
	importPath := `pkg/test`
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
	if typ, ok := tg.cache[expr]; ok {
		return typ
	}

	f := tg.tf.Parse(`eval`, "package testcase\nvar _ "+expr)
	config := &types.Config{
		Context:  tg.tf.Context,
		Sizes:    &types.StdSizes{WordSize: 4, MaxAlign: 8},
		Importer: tg.tf,
	}
	pkg := tg.tf.Packages[`pkg/test`]
	ck := types.NewChecker(config, tg.tf.FileSet, pkg, tg.tf.Info)
	if err := ck.Files([]*ast.File{f}); err != nil {
		tg.tf.T.Fatalf("failed to type check expression %q: %v", expr, err)
	}

	node := f.Decls[0].(*ast.GenDecl).Specs[0].(*ast.ValueSpec).Type
	typ := tg.tf.Info.Types[node].Type
	tg.cache[expr] = typ
	return typ
}

func (tg typeGetter) TypeList(expr ...string) typesutil.TypeList {
	tg.tf.T.Helper()
	result := make([]types.Type, len(expr))
	for i, expr := range expr {
		result[i] = tg.Type(expr)
	}
	return result
}
