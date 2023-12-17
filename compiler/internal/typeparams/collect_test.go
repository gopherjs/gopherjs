package typeparams

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/gopherjs/gopherjs/internal/srctesting"
	"golang.org/x/tools/go/ast/astutil"
)

func TestVisitor(t *testing.T) {
	// This test verifies that instance collector is able to discover
	// instantiations of generic types and functions in all possible contexts.
	const src = `package testcase

	type A struct{}
	type B struct{}
	type C struct{}
	type D struct{}
	type E struct{}
	type F struct{}
	type G struct{}

	type typ[T any, V any] []T
	func (t *typ[T, V]) method(x T) {}
	func fun[U any, W any](x U, y W) {}

	func entry1(arg typ[int8, A]) (result typ[int16, A]) {
		fun(1, A{})
		fun[int8, A](1, A{})
		println(fun[int16, A])

		t := typ[int, A]{}
		t.method(0)
		(*typ[int32, A]).method(nil, 0)
		type x struct{ T []typ[int64, A] }

		return
	}

	func entry2[T any](arg typ[int8, T]) (result typ[int16, T]) {
		var zeroT T
		fun(1, zeroT)
		fun[int8, T](1, zeroT)
		println(fun[int16, T])

		t := typ[int, T]{}
		t.method(0)
		(*typ[int32, T]).method(nil, 0)
		type x struct{ T []typ[int64, T] }

		return
	}

	type entry3[T any] struct{
		typ[int, T]
		field1 struct { field2 typ[int8, T] }
	}
	func (e entry3[T]) method(arg typ[int8, T]) (result typ[int16, T]) {
		var zeroT T
		fun(1, zeroT)
		fun[int8, T](1, zeroT)
		println(fun[int16, T])

		t := typ[int, T]{}
		t.method(0)
		(*typ[int32, T]).method(nil, 0)
		type x struct{ T []typ[int64, T] }

		return
	}

	type entry4 struct{
		typ[int, E]
		field1 struct { field2 typ[int8, E] }
	}

	type entry5 = typ[int, F]
	`
	fset := token.NewFileSet()
	file := srctesting.Parse(t, fset, src)
	info, pkg := srctesting.Check(t, fset, file)

	lookupObj := func(name string) types.Object {
		parts := strings.Split(name, ".")
		obj := pkg.Scope().Lookup(parts[0])
		if len(parts) == 1 {
			return obj
		}
		obj, _, _ = types.LookupFieldOrMethod(obj.Type(), true, obj.Pkg(), parts[1])
		return obj
	}
	lookupType := func(name string) types.Type { return lookupObj(name).Type() }
	lookupDecl := func(name string) ast.Node {
		obj := lookupObj(name)
		path, _ := astutil.PathEnclosingInterval(file, obj.Pos(), obj.Pos())
		for _, n := range path {
			switch n.(type) {
			case *ast.FuncDecl, *ast.TypeSpec:
				return n
			}
		}
		t.Fatalf("Could not find AST node representing %v", obj)
		return nil
	}

	// Generates a list of instances we expect to discover from functions and
	// methods. Sentinel type is a type parameter we use uniquely within one
	// context, which allows us to make sure that collection is not being tested
	// against a wrong part of AST.
	instancesInFunc := func(sentinel types.Type) []Instance {
		return []Instance{
			{
				// Called with type arguments inferred.
				Object: lookupObj("fun"),
				TArgs:  []types.Type{types.Typ[types.Int], sentinel},
			}, {
				// Called with type arguments explicitly specified.
				Object: lookupObj("fun"),
				TArgs:  []types.Type{types.Typ[types.Int8], sentinel},
			}, {
				// Passed as an argument.
				Object: lookupObj("fun"),
				TArgs:  []types.Type{types.Typ[types.Int16], sentinel},
			}, {
				// Literal expression.
				Object: lookupObj("typ"),
				TArgs:  []types.Type{types.Typ[types.Int], sentinel},
			}, {
				// Function argument.
				Object: lookupObj("typ"),
				TArgs:  []types.Type{types.Typ[types.Int8], sentinel},
			}, {
				// Function return type.
				Object: lookupObj("typ"),
				TArgs:  []types.Type{types.Typ[types.Int16], sentinel},
			}, {
				// Method expression.
				Object: lookupObj("typ"),
				TArgs:  []types.Type{types.Typ[types.Int32], sentinel},
			}, {
				// Type decl statement.
				Object: lookupObj("typ"),
				TArgs:  []types.Type{types.Typ[types.Int64], sentinel},
			},
		}
	}

	// Generates a list of instances we expect to discover from type declarations.
	// Sentinel type is a type parameter we use uniquely within one context, which
	// allows us to make sure that collection is not being tested against a wrong
	// part of AST.
	instancesInType := func(sentinel types.Type) []Instance {
		return []Instance{
			{
				Object: lookupObj("typ"),
				TArgs:  []types.Type{types.Typ[types.Int], sentinel},
			}, {
				Object: lookupObj("typ"),
				TArgs:  []types.Type{types.Typ[types.Int8], sentinel},
			},
		}
	}

	tests := []struct {
		descr    string
		resolver *Resolver
		node     ast.Node
		want     []Instance
	}{
		{
			descr:    "non-generic function",
			resolver: nil,
			node:     lookupDecl("entry1"),
			want:     instancesInFunc(lookupType("A")),
		}, {
			descr: "generic function",
			resolver: NewResolver(
				types.NewContext(),
				ToSlice(lookupType("entry2").(*types.Signature).TypeParams()),
				[]types.Type{lookupType("B")},
			),
			node: lookupDecl("entry2"),
			want: instancesInFunc(lookupType("B")),
		}, {
			descr: "generic method",
			resolver: NewResolver(
				types.NewContext(),
				ToSlice(lookupType("entry3.method").(*types.Signature).RecvTypeParams()),
				[]types.Type{lookupType("C")},
			),
			node: lookupDecl("entry3.method"),
			want: append(
				instancesInFunc(lookupType("C")),
				Instance{
					Object: lookupObj("entry3"),
					TArgs:  []types.Type{lookupType("C")},
				},
			),
		}, {
			descr: "generic type declaration",
			resolver: NewResolver(
				types.NewContext(),
				ToSlice(lookupType("entry3").(*types.Named).TypeParams()),
				[]types.Type{lookupType("D")},
			),
			node: lookupDecl("entry3"),
			want: instancesInType(lookupType("D")),
		}, {
			descr:    "non-generic type declaration",
			resolver: nil,
			node:     lookupDecl("entry4"),
			want:     instancesInType(lookupType("E")),
		}, {
			descr:    "non-generic type alias",
			resolver: nil,
			node:     lookupDecl("entry5"),
			want: []Instance{
				{
					Object: lookupObj("typ"),
					TArgs:  []types.Type{types.Typ[types.Int], lookupType("F")},
				},
			},
		},
	}

	for _, test := range tests {
		v := visitor{
			instances: &InstanceSet{},
			resolver:  test.resolver,
			info:      info,
		}
		ast.Walk(&v, test.node)
		got := v.instances.Values()
		if diff := cmp.Diff(test.want, got, instanceOpts()); diff != "" {
			t.Errorf("Discovered instance diff (-want,+got):\n%s", diff)
		}
	}
}

func TestSeedVisitor(t *testing.T) {
	src := `package test
	type typ[T any] int
	func (t typ[T]) method(arg T) { var x typ[string]; _ = x }
	func fun[T any](arg T) { var y typ[string]; _ = y }

	const a typ[int] = 1
	var b typ[int]
	type c struct { field typ[int8] }
	func (_ c) method() { var _ typ[int16] }
	type d = typ[int32]
	func e() { var _ typ[int64] }
	`

	fset := token.NewFileSet()
	file := srctesting.Parse(t, fset, src)
	info, pkg := srctesting.Check(t, fset, file)

	sv := seedVisitor{
		visitor: visitor{
			instances: &InstanceSet{},
			resolver:  nil,
			info:      info,
		},
		objMap: map[types.Object]ast.Node{},
	}
	ast.Walk(&sv, file)

	inst := func(tArg types.Type) Instance {
		return Instance{
			Object: pkg.Scope().Lookup("typ"),
			TArgs:  []types.Type{tArg},
		}
	}
	want := []Instance{
		inst(types.Typ[types.Int]),
		inst(types.Typ[types.Int8]),
		inst(types.Typ[types.Int16]),
		inst(types.Typ[types.Int32]),
		inst(types.Typ[types.Int64]),
	}
	got := sv.instances.Values()
	if diff := cmp.Diff(want, got, instanceOpts()); diff != "" {
		t.Errorf("Instances from initialSeeder contain diff (-want,+got):\n%s", diff)
	}
}

func TestCollector(t *testing.T) {
	src := `package test
	type typ[T any] int
	func (t typ[T]) method(arg T) { var _ typ[int]; fun[int8](0) }
	func fun[T any](arg T) { var _ typ[int16] }

	type ignore = int

	func a() {
		var _ typ[int32]
		fun[int64](0)
	}
	`

	fset := token.NewFileSet()
	file := srctesting.Parse(t, fset, src)
	info, pkg := srctesting.Check(t, fset, file)

	c := Collector{
		TContext:  types.NewContext(),
		Info:      info,
		Instances: &InstanceSet{},
	}
	c.Scan(file)

	inst := func(name string, tArg types.Type) Instance {
		return Instance{
			Object: pkg.Scope().Lookup(name),
			TArgs:  []types.Type{tArg},
		}
	}
	want := []Instance{
		inst("typ", types.Typ[types.Int]),
		inst("fun", types.Typ[types.Int8]),
		inst("typ", types.Typ[types.Int16]),
		inst("typ", types.Typ[types.Int32]),
		inst("fun", types.Typ[types.Int64]),
	}
	got := c.Instances.Values()
	if diff := cmp.Diff(want, got, instanceOpts()); diff != "" {
		t.Errorf("Instances from initialSeeder contain diff (-want,+got):\n%s", diff)
	}
}
