package grouper

import (
	"go/ast"
	"go/types"
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
		// func() testData {
		// 	tg := readTypes(t, `
		// 		type Foo struct {}
		// 		type Bar struct {}
		// 		var m map[Bar]Foo`)
		// 	return testData{
		// 		name:    `depend on map key and element`,
		// 		context: tg.tf.Context,
		// 		instance: typeparams.Instance{
		// 			Object: tg.Object(`m`),
		// 		},
		// 		expName: tg.Type(`map[Bar]Foo`),
		// 		expDeps: tg.TypeList(`Bar`, `Foo`),
		// 	}
		// }(),
		// func() testData {
		// 	tg := readTypes(t, `
		// 		type Foo struct { X Bar }
		// 		type Bar struct {}`)
		// 	return testData{
		// 		name:    `do not need to depend on fields`,
		// 		context: tg.tf.Context,
		// 		instance: typeparams.Instance{
		// 			Object: tg.Object(`Foo`),
		// 		},
		// 		expName: tg.Type(`Foo`),
		// 		expDeps: nil,
		// 	}
		// }(),
		// func() testData {
		// 	tg := readTypes(t, `
		// 		type Foo struct {}
		// 		func (f Foo) Bar(x int, y int) {}`)
		// 	return testData{
		// 		name:    `depend on receiver type and do not type methods`,
		// 		context: tg.tf.Context,
		// 		instance: typeparams.Instance{
		// 			Object: tg.Object(`Foo.Bar`),
		// 		},
		// 		expName: nil,
		// 		expDeps: tg.TypeList(`Foo`),
		// 	}
		// }(),
		// func() testData {
		// 	tg := readTypes(t, `
		// 		type Foo struct {}
		// 		func (f *Foo) Bar(x int, y int) {}`)
		// 	return testData{
		// 		name:    `depend on receiver type without the pointer`,
		// 		context: tg.tf.Context,
		// 		instance: typeparams.Instance{
		// 			Object: tg.Object(`Foo.Bar`),
		// 		},
		// 		expName: nil,
		// 		expDeps: tg.TypeList(`Foo`),
		// 	}
		// }(),
		// func() testData {
		// 	tg := readTypes(t, `
		// 		type Foo[T any] struct {}
		// 		func Bar[T any](x *Foo[T]) []*Foo[T] { return nil }
		// 		type Baz struct {}`)
		// 	return testData{
		// 		name:    `depend on type arguments but not parameters nor results`,
		// 		context: tg.tf.Context,
		// 		instance: typeparams.Instance{
		// 			Object: tg.Object(`Bar`),
		// 			TArgs:  tg.TypeList(`Baz`),
		// 		},
		// 		expName: nil,
		// 		expDeps: tg.TypeList(`Baz`),
		// 	}
		// }(),
		// func() testData {
		// 	tg := readTypes(t, `
		// 		type Foo[T any] struct {}
		// 		type Bar struct {}
		// 		var Baz = Foo[Bar]{}`)
		// 	return testData{
		// 		name:    `variables get typed and depend on their type parts`,
		// 		context: tg.tf.Context,
		// 		instance: typeparams.Instance{
		// 			Object: tg.Object(`Baz`),
		// 		},
		// 		expName: tg.Type(`Foo[Bar]`),
		// 		expDeps: tg.TypeList(`Bar`),
		// 	}
		// }(),
		// func() testData {
		// 	tg := readTypes(t, `
		// 		var Foo []struct{}`)
		// 	return testData{
		// 		name:    `do not depend on empty structs`,
		// 		context: tg.tf.Context,
		// 		instance: typeparams.Instance{
		// 			Object: tg.Object(`Foo`),
		// 		},
		// 		expName: tg.Type(`[]struct{}`),
		// 		expDeps: nil,
		// 	}
		// }(),
		// func() testData {
		// 	tg := readTypes(t, `
		// 		func Foo[T any]() any {
		// 			type Bar struct{ x T}
		// 			return Bar{}
		// 		}
		// 		type Baz struct{}`)
		// 	return testData{
		// 		name:    `depend on implicit nesting type arguments`,
		// 		context: tg.tf.Context,
		// 		instance: typeparams.Instance{
		// 			Object: tg.Object(`Foo.Bar`),
		// 			TNest:  tg.TypeList(`Baz`),
		// 		},
		// 		expName: tg.Object(`Foo.Bar`).Type(),
		// 		expDeps: tg.TypeList(`Baz`),
		// 	}
		// }(),
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			info := &Info{}
			// Instead of calling SetInstance, we manually set the type and dependencies
			// so that we can tell it to not skip the same package dependencies.
			info.setType(test.context, test.instance)
			info.setAllDeps(test.context, test.instance, false)

			if info.name != test.expName {
				t.Errorf("expected type %v, got %v", test.expName, info.name)
			}
			if diff := cmp.Diff(test.expDeps, info.dep); diff != "" {
				t.Errorf("unexpected dependencies (-want +got):\n%s", diff)
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

func (tg typeGetter) Named(expr string) *types.Named {
	tg.tf.T.Helper()
	return tg.Type(expr).(*types.Named)
}

func (tg typeGetter) NamedSet(exprs ...string) map[*types.Named]struct{} {
	tg.tf.T.Helper()
	result := make(map[*types.Named]struct{}, len(exprs))
	for _, expr := range exprs {
		result[tg.Named(expr)] = struct{}{}
	}
	return result
}
