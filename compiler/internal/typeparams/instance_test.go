package typeparams

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/gopherjs/gopherjs/internal/srctesting"
	"github.com/gopherjs/gopherjs/internal/testingx"
)

func instanceOpts() cmp.Options {
	return cmp.Options{
		// Instances are represented by their IDs for diffing purposes.
		cmp.Transformer("Instance", func(i Instance) string {
			return i.String()
		}),
		// Order of instances in a slice doesn't matter, sort them by ID.
		cmpopts.SortSlices(func(a, b Instance) bool {
			return a.String() < b.String()
		}),
	}
}

func TestInstanceString(t *testing.T) {
	const src = `package testcase

	type Ints []int

	type Typ[T any, V any] []T
	func (t Typ[T, V]) Method(x T) {}

	type typ[T any, V any] []T
	func (t typ[T, V]) method(x T) {}

	func Fun[U any, W any](x, y U) {}
	func fun[U any, W any](x, y U) {}
	`
	fset := token.NewFileSet()
	_, pkg := srctesting.Check(t, fset, srctesting.Parse(t, fset, src))
	mustType := testingx.Must[types.Type](t)

	tests := []struct {
		descr          string
		instance       Instance
		wantStr        string
		wantTypeString string
	}{{
		descr: "exported type",
		instance: Instance{
			Object: pkg.Scope().Lookup("Typ"),
			TArgs:  []types.Type{types.Typ[types.Int], types.Typ[types.String]},
		},
		wantStr:        "test.Typ<int, string>",
		wantTypeString: "testcase.Typ[int, string]",
	}, {
		descr: "exported method",
		instance: Instance{
			Object: pkg.Scope().Lookup("Typ").Type().(*types.Named).Method(0),
			TArgs:  []types.Type{types.Typ[types.Int], types.Typ[types.String]},
		},
		wantStr: "test.Typ.Method<int, string>",
	}, {
		descr: "exported function",
		instance: Instance{
			Object: pkg.Scope().Lookup("Fun"),
			TArgs:  []types.Type{types.Typ[types.Int], types.Typ[types.String]},
		},
		wantStr: "test.Fun<int, string>",
	}, {
		descr: "unexported type",
		instance: Instance{
			Object: pkg.Scope().Lookup("typ"),
			TArgs:  []types.Type{types.Typ[types.Int], types.Typ[types.String]},
		},
		wantStr:        "test.typ<int, string>",
		wantTypeString: "testcase.typ[int, string]",
	}, {
		descr: "unexported method",
		instance: Instance{
			Object: pkg.Scope().Lookup("typ").Type().(*types.Named).Method(0),
			TArgs:  []types.Type{types.Typ[types.Int], types.Typ[types.String]},
		},
		wantStr: "test.typ.method<int, string>",
	}, {
		descr: "unexported function",
		instance: Instance{
			Object: pkg.Scope().Lookup("fun"),
			TArgs:  []types.Type{types.Typ[types.Int], types.Typ[types.String]},
		},
		wantStr: "test.fun<int, string>",
	}, {
		descr: "no type params",
		instance: Instance{
			Object: pkg.Scope().Lookup("Ints"),
		},
		wantStr:        "test.Ints",
		wantTypeString: "testcase.Ints",
	}, {
		descr: "complex parameter type",
		instance: Instance{
			Object: pkg.Scope().Lookup("fun"),
			TArgs: []types.Type{
				types.NewSlice(types.Typ[types.Int]),
				mustType(types.Instantiate(nil, pkg.Scope().Lookup("typ").Type(), []types.Type{
					types.Typ[types.Int],
					types.Typ[types.String],
				}, true)),
			},
		},
		wantStr: "test.fun<[]int, test.typ[int, string]>",
	}}

	for _, test := range tests {
		t.Run(test.descr, func(t *testing.T) {
			got := test.instance.String()
			if got != test.wantStr {
				t.Errorf("Got: instance string %q. Want: %q.", got, test.wantStr)
			}
			if test.wantTypeString != "" {
				got = test.instance.TypeString()
				if got != test.wantTypeString {
					t.Errorf("Got: instance type string %q. Want: %q.", got, test.wantTypeString)
				}
			}
		})
	}
}

func TestInstanceQueue(t *testing.T) {
	const src = `package test
	type Typ[T any, V any] []T
	func Fun[U any, W any](x, y U) {}
	`
	fset := token.NewFileSet()
	_, pkg := srctesting.Check(t, fset, srctesting.Parse(t, fset, src))

	i1 := Instance{
		Object: pkg.Scope().Lookup("Typ"),
		TArgs:  []types.Type{types.Typ[types.String], types.Typ[types.String]},
	}
	i2 := Instance{
		Object: pkg.Scope().Lookup("Typ"),
		TArgs:  []types.Type{types.Typ[types.Int], types.Typ[types.Int]},
	}
	i3 := Instance{
		Object: pkg.Scope().Lookup("Fun"),
		TArgs:  []types.Type{types.Typ[types.String], types.Typ[types.String]},
	}

	set := InstanceSet{}
	set.Add(i1, i2)

	if ex := set.exhausted(); ex {
		t.Errorf("Got: set.exhausted() = true. Want: false")
	}

	gotValues := set.Values()
	wantValues := []Instance{i1, i2}
	if diff := cmp.Diff(wantValues, gotValues, instanceOpts()); diff != "" {
		t.Errorf("set.Values() returned diff (-want,+got):\n%s", diff)
	}

	p1, ok := set.next()
	if !ok {
		t.Errorf("Got: _, ok := set.next(); ok == false. Want: true.")
	}
	p2, ok := set.next()
	if !ok {
		t.Errorf("Got: _, ok := set.next(); ok == false. Want: true.")
	}
	if ex := set.exhausted(); !ex {
		t.Errorf("Got: set.exhausted() = false. Want: true")
	}

	_, ok = set.next()
	if ok {
		t.Errorf("Got: _, ok := set.next(); ok == true. Want: false.")
	}

	set.Add(i1) // Has been enqueued before.
	if ex := set.exhausted(); !ex {
		t.Errorf("Got: set.exhausted() = false. Want: true")
	}

	set.Add(i3)
	p3, ok := set.next()
	if !ok {
		t.Errorf("Got: _, ok := set.next(); ok == false. Want: true.")
	}

	added := []Instance{i1, i2, i3}
	processed := []Instance{p1, p2, p3}

	diff := cmp.Diff(added, processed, instanceOpts())
	if diff != "" {
		t.Errorf("Processed instances differ from added (-want,+got):\n%s", diff)
	}

	gotValues = set.Values()
	wantValues = []Instance{i1, i2, i3}
	if diff := cmp.Diff(wantValues, gotValues, instanceOpts()); diff != "" {
		t.Errorf("set.Values() returned diff (-want,+got):\n%s", diff)
	}
}
