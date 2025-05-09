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
		type y[X any] struct{ T []typ[A, X] }
		_ = y[int8]{}
		_ = y[A]{}

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
		type y[X any] struct{ T []typ[T, X] }
		_ = y[int8]{}
		_ = y[T]{}

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
		type y[X any] struct{ T []typ[T, X] }
		_ = y[int8]{}
		_ = y[T]{}

		return
	}

	type entry4 struct{
		typ[int, E]
		field1 struct { field2 typ[int8, E] }
	}

	type entry5 = typ[int, F]
	`
	f := srctesting.New(t)
	file := f.Parse("test.go", src)
	info, pkg := f.Check("pkg/test", file)

	lookupObj := func(name string) types.Object {
		obj := srctesting.LookupObj(pkg, name)
		if obj == nil {
			t.Fatalf("Object %q not found", name)
		}
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
				Object: lookupObj("typ.method"),
				TArgs:  []types.Type{types.Typ[types.Int], sentinel},
			}, {
				// Function argument.
				Object: lookupObj("typ"),
				TArgs:  []types.Type{types.Typ[types.Int8], sentinel},
			}, {
				Object: lookupObj("typ.method"),
				TArgs:  []types.Type{types.Typ[types.Int8], sentinel},
			}, {
				// Function return type.
				Object: lookupObj("typ"),
				TArgs:  []types.Type{types.Typ[types.Int16], sentinel},
			}, {
				Object: lookupObj("typ.method"),
				TArgs:  []types.Type{types.Typ[types.Int16], sentinel},
			}, {
				// Method expression.
				Object: lookupObj("typ"),
				TArgs:  []types.Type{types.Typ[types.Int32], sentinel},
			}, {
				Object: lookupObj("typ.method"),
				TArgs:  []types.Type{types.Typ[types.Int32], sentinel},
			}, {
				// Type decl statement.
				Object: lookupObj("typ"),
				TArgs:  []types.Type{types.Typ[types.Int64], sentinel},
			}, {
				Object: lookupObj("typ.method"),
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
				Object: lookupObj("typ.method"),
				TArgs:  []types.Type{types.Typ[types.Int], sentinel},
			}, {
				Object: lookupObj("typ"),
				TArgs:  []types.Type{types.Typ[types.Int8], sentinel},
			}, {
				Object: lookupObj("typ.method"),
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
			want: append(
				instancesInFunc(lookupType("A")),
				Instance{
					Object: lookupObj("entry1.y"),
					TArgs:  []types.Type{types.Typ[types.Int8]},
				},
				Instance{
					Object: lookupObj("entry1.y"),
					TArgs:  []types.Type{lookupType("A")},
				},
			),
		}, {
			descr: "generic function",
			resolver: NewResolver(
				types.NewContext(),
				lookupType("entry2").(*types.Signature).TypeParams(),
				[]types.Type{lookupType("B")},
				nil,
			),
			node: lookupDecl("entry2"),
			want: append(
				instancesInFunc(lookupType("B")),
				Instance{
					Object: lookupObj("entry2.x"),
					TNest:  []types.Type{lookupType("B")},
				},
				Instance{
					Object: lookupObj("entry1.y"),
					TNest:  []types.Type{lookupType("B")},
					TArgs:  []types.Type{types.Typ[types.Int8]},
				},
				Instance{
					Object: lookupObj("entry2.y"),
					TNest:  []types.Type{lookupType("B")},
					TArgs:  []types.Type{lookupType("B")},
				},
			),
		}, {
			descr: "generic method",
			resolver: NewResolver(
				types.NewContext(),
				lookupType("entry3.method").(*types.Signature).RecvTypeParams(),
				[]types.Type{lookupType("C")},
				nil,
			),
			node: lookupDecl("entry3.method"),
			want: append(
				instancesInFunc(lookupType("C")),
				Instance{
					Object: lookupObj("entry3"),
					TArgs:  []types.Type{lookupType("C")},
				},
				Instance{
					Object: lookupObj("entry3.method"),
					TArgs:  []types.Type{lookupType("C")},
				},
				Instance{
					Object: lookupObj("entry3.method.x"),
					TNest:  []types.Type{lookupType("C")},
				},
				Instance{
					Object: lookupObj("entry3.method.y"),
					TNest:  []types.Type{lookupType("C")},
					TArgs:  []types.Type{types.Typ[types.Int8]},
				},
				Instance{
					Object: lookupObj("entry3.method.y"),
					TNest:  []types.Type{lookupType("C")},
					TArgs:  []types.Type{lookupType("C")},
				},
			),
		}, {
			descr: "generic type declaration",
			resolver: NewResolver(
				types.NewContext(),
				lookupType("entry3").(*types.Named).TypeParams(),
				[]types.Type{lookupType("D")},
				nil,
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
				{
					Object: lookupObj("typ.method"),
					TArgs:  []types.Type{types.Typ[types.Int], lookupType("F")},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.descr, func(t *testing.T) {
			v := visitor{
				instances: &PackageInstanceSets{},
				resolver:  test.resolver,
				info:      info,
			}
			ast.Walk(&v, test.node)
			got := v.instances.Pkg(pkg).Values()
			if diff := cmp.Diff(test.want, got, instanceOpts()); diff != "" {
				t.Errorf("Discovered instance diff (-want,+got):\n%s", diff)
			}
		})
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

	f := srctesting.New(t)
	file := f.Parse("test.go", src)
	info, pkg := f.Check("pkg/test", file)

	sv := seedVisitor{
		visitor: visitor{
			instances: &PackageInstanceSets{},
			resolver:  nil,
			info:      info,
		},
		objMap: map[types.Object]ast.Node{},
	}
	ast.Walk(&sv, file)

	tInst := func(tArg types.Type) Instance {
		return Instance{
			Object: pkg.Scope().Lookup("typ"),
			TArgs:  []types.Type{tArg},
		}
	}
	mInst := func(tArg types.Type) Instance {
		return Instance{
			Object: srctesting.LookupObj(pkg, "typ.method"),
			TArgs:  []types.Type{tArg},
		}
	}
	want := []Instance{
		tInst(types.Typ[types.Int]),
		mInst(types.Typ[types.Int]),
		tInst(types.Typ[types.Int8]),
		mInst(types.Typ[types.Int8]),
		tInst(types.Typ[types.Int16]),
		mInst(types.Typ[types.Int16]),
		tInst(types.Typ[types.Int32]),
		mInst(types.Typ[types.Int32]),
		tInst(types.Typ[types.Int64]),
		mInst(types.Typ[types.Int64]),
	}
	got := sv.instances.Pkg(pkg).Values()
	if diff := cmp.Diff(want, got, instanceOpts()); diff != "" {
		t.Errorf("Instances from initialSeeder contain diff (-want,+got):\n%s", diff)
	}
}

func TestCollector(t *testing.T) {
	src := `package test
	type typ[T any] int
	func (t typ[T]) method(arg T) { var _ typ[int]; fun[int8](0) }
	func fun[T any](arg T) {
		var _ typ[int16]

		type nested[U any] struct{}
		_ = nested[T]{}
	}

	type ignore = int

	func a() {
		var _ typ[int32]
		fun[int64](0)
	}
	`

	f := srctesting.New(t)
	file := f.Parse("test.go", src)
	info, pkg := f.Check("pkg/test", file)

	c := Collector{
		TContext:  types.NewContext(),
		Info:      info,
		Instances: &PackageInstanceSets{},
	}
	c.Scan(pkg, file)

	inst := func(name, tNest, tArg string) Instance {
		return createInstance(t, pkg, f.FileSet, name, tNest, tArg)
	}
	want := []Instance{
		inst(`typ`, ``, `int`),
		inst(`typ.method`, ``, `int`),
		inst(`fun`, ``, `int8`),
		inst(`fun.nested`, `int8`, `int8`),
		inst(`typ`, ``, `int16`),
		inst(`typ.method`, ``, `int16`),
		inst(`typ`, ``, `int32`),
		inst(`typ.method`, ``, `int32`),
		inst(`fun`, ``, `int64`),
		inst(`fun.nested`, `int64`, `int64`),
	}
	got := c.Instances.Pkg(pkg).Values()
	if diff := cmp.Diff(want, got, instanceOpts()); diff != "" {
		t.Errorf("Instances from initialSeeder contain diff (-want,+got):\n%s", diff)
	}
}

func TestCollector_MoreNesting(t *testing.T) {
	src := `package test

	func fun[T any]() {
		type nestedCon struct{ X T }
		_ = nestedCon{}

		type nestedGen[U any] struct{ Y T; Z U }
		_ = nestedGen[T]{}
		_ = nestedGen[int8]{}

		type nestedCover[T any] struct{ W T }
		_ = nestedCover[T]{}
		_ = nestedCover[int16]{}
	}

	func a() {
		fun[int32]()
		fun[int64]()
	}
	`

	f := srctesting.New(t)
	file := f.Parse(`test.go`, src)
	info, pkg := f.Check(`pkg/test`, file)

	c := Collector{
		TContext:  types.NewContext(),
		Info:      info,
		Instances: &PackageInstanceSets{},
	}
	c.Scan(pkg, file)

	inst := func(name, tNest, tArg string) Instance {
		return createInstance(t, pkg, f.FileSet, name, tNest, tArg)
	}
	want := []Instance{
		inst(`fun`, ``, `int32`),
		inst(`fun`, ``, `int64`),

		inst(`fun.nestedCon`, `int32`, ``),
		inst(`fun.nestedCon`, `int64`, ``),

		inst(`fun.nestedGen`, `int32`, `int32`),
		inst(`fun.nestedGen`, `int32`, `int8`),
		inst(`fun.nestedGen`, `int64`, `int64`),
		inst(`fun.nestedGen`, `int64`, `int8`),

		inst(`fun.nestedCover`, `int32`, `int32`),
		inst(`fun.nestedCover`, `int32`, `int16`),
		inst(`fun.nestedCover`, `int64`, `int64`),
		inst(`fun.nestedCover`, `int64`, `int16`),
	}
	got := c.Instances.Pkg(pkg).Values()
	if diff := cmp.Diff(want, got, instanceOpts()); diff != `` {
		t.Errorf("Instances from initialSeeder contain diff (-want,+got):\n%s", diff)
	}
}

func TestCollector_NestingWithVars(t *testing.T) {
	// This is loosely based off of go1.19.13/test/typeparam/issue47740b.go
	// I was getting an error where `Q.print[int;]` was showing up when
	// `Q.print` is not in a nesting context with `int` and this helped debug
	// it. The problem was that `q` was being treated like a type not a var.
	src := `package test

	type Q struct{ v any }
	func (q Q) print() {
		println(q.v)
	}

	func newQ(v any) Q {
		return Q{v}
	}

	type S[T any] struct{ x T }
	func (s S[T]) echo() {
		q := newQ(s.x)
		q.print()
	}

	func a() {
		s := S[int]{x: 0}
		s.echo()
	}
	`

	f := srctesting.New(t)
	file := f.Parse(`test.go`, src)
	info, pkg := f.Check(`pkg/test`, file)

	c := Collector{
		TContext:  types.NewContext(),
		Info:      info,
		Instances: &PackageInstanceSets{},
	}
	c.Scan(pkg, file)

	inst := func(name, tNest, tArg string) Instance {
		return createInstance(t, pkg, f.FileSet, name, tNest, tArg)
	}
	want := []Instance{
		inst(`S`, ``, `int`),
		inst(`S.echo`, ``, `int`),
	}
	got := c.Instances.Pkg(pkg).Values()
	if diff := cmp.Diff(want, got, instanceOpts()); diff != `` {
		t.Errorf("Instances from initialSeeder contain diff (-want,+got):\n%s", diff)
	}
}

func createInstance(t *testing.T, pkg *types.Package, fSet *token.FileSet, name, tNest, tArg string) Instance {
	obj := srctesting.LookupObj(pkg, name)
	if obj == nil {
		t.Fatalf(`Object %q not found in package %q`, name, pkg.Name())
	}
	return Instance{
		Object: srctesting.LookupObj(pkg, name),
		TNest:  evalTypeArgs(t, fSet, pkg, tNest),
		TArgs:  evalTypeArgs(t, fSet, pkg, tArg),
	}
}

func evalTypeArgs(t *testing.T, fSet *token.FileSet, pkg *types.Package, expr string) []types.Type {
	if len(expr) == 0 {
		return nil
	}
	args := strings.Split(expr, ",")
	targs := make([]types.Type, 0, len(args))
	for _, astr := range args {
		tv, err := types.Eval(fSet, pkg, 0, astr)
		if err != nil {
			t.Fatalf("Eval(%s) failed: %v", astr, err)
		}
		targs = append(targs, tv.Type)
	}
	return targs
}

func TestCollector_CrossPackage(t *testing.T) {
	f := srctesting.New(t)
	const src = `package foo
	type X[T any] struct {Value T}

	func F[G any](g G) {
		x := X[G]{}
		println(x)
	}

	func DoFoo() {
		F(int8(8))
	}
	`
	fooFile := f.Parse("foo.go", src)
	_, fooPkg := f.Check("pkg/foo", fooFile)

	const src2 = `package bar
	import "pkg/foo"
	func FProxy[T any](t T) {
		foo.F[T](t)
	}
	func DoBar() {
		FProxy(int16(16))
	}
	`
	barFile := f.Parse("bar.go", src2)
	_, barPkg := f.Check("pkg/bar", barFile)

	c := Collector{
		TContext:  types.NewContext(),
		Info:      f.Info,
		Instances: &PackageInstanceSets{},
	}
	c.Scan(barPkg, barFile)
	c.Scan(fooPkg, fooFile)

	inst := func(pkg *types.Package, name string, tArg types.BasicKind) Instance {
		return Instance{
			Object: srctesting.LookupObj(pkg, name),
			TArgs:  []types.Type{types.Typ[tArg]},
		}
	}

	wantFooInstances := []Instance{
		inst(fooPkg, "F", types.Int16), // Found in "pkg/foo".
		inst(fooPkg, "F", types.Int8),
		inst(fooPkg, "X", types.Int16), // Found due to F[int16] found in "pkg/foo".
		inst(fooPkg, "X", types.Int8),
	}
	gotFooInstances := c.Instances.Pkg(fooPkg).Values()
	if diff := cmp.Diff(wantFooInstances, gotFooInstances, instanceOpts()); diff != "" {
		t.Errorf("Instances from pkg/foo contain diff (-want,+got):\n%s", diff)
	}

	wantBarInstances := []Instance{
		inst(barPkg, "FProxy", types.Int16),
	}
	gotBarInstances := c.Instances.Pkg(barPkg).Values()
	if diff := cmp.Diff(wantBarInstances, gotBarInstances, instanceOpts()); diff != "" {
		t.Errorf("Instances from pkg/foo contain diff (-want,+got):\n%s", diff)
	}
}

func TestResolver_SubstituteSelection(t *testing.T) {
	tests := []struct {
		descr   string
		src     string
		wantObj string
		wantSig string
	}{{
		descr: "type parameter method",
		src: `package test
		type stringer interface{ String() string }

		type x struct{}
		func (_ x) String() string { return "" }

		type g[T stringer] struct{}
		func (_ g[T]) Method(t T) string {
			return t.String()
		}`,
		wantObj: "func (pkg/test.x).String() string",
		wantSig: "func() string",
	}, {
		descr: "generic receiver type with type parameter",
		src: `package test
			type x struct{}

			type g[T any] struct{}
			func (_ g[T]) Method(t T) string {
				return g[T]{}.Method(t)
			}`,
		wantObj: "func (pkg/test.g[pkg/test.x]).Method(t pkg/test.x) string",
		wantSig: "func(t pkg/test.x) string",
	}, {
		descr: "method expression",
		src: `package test
				type x struct{}

				type g[T any] struct{}
				func (recv g[T]) Method(t T) string {
					return g[T].Method(recv, t)
				}`,
		wantObj: "func (pkg/test.g[pkg/test.x]).Method(t pkg/test.x) string",
		wantSig: "func(recv pkg/test.g[pkg/test.x], t pkg/test.x) string",
	}}

	for _, test := range tests {
		t.Run(test.descr, func(t *testing.T) {
			f := srctesting.New(t)
			file := f.Parse("test.go", test.src)
			info, pkg := f.Check("pkg/test", file)

			method := srctesting.LookupObj(pkg, "g.Method").(*types.Func).Type().(*types.Signature)
			resolver := NewResolver(nil, method.RecvTypeParams(), []types.Type{srctesting.LookupObj(pkg, "x").Type()}, nil)

			if l := len(info.Selections); l != 1 {
				t.Fatalf("Got: %d selections. Want: 1", l)
			}
			for _, sel := range info.Selections {
				gotObj := types.ObjectString(resolver.SubstituteSelection(sel).Obj(), nil)
				if gotObj != test.wantObj {
					t.Fatalf("Got: resolver.SubstituteSelection().Obj() = %q. Want: %q.", gotObj, test.wantObj)
				}
				gotSig := types.TypeString(resolver.SubstituteSelection(sel).Type(), nil)
				if gotSig != test.wantSig {
					t.Fatalf("Got: resolver.SubstituteSelection().Type() = %q. Want: %q.", gotSig, test.wantSig)
				}
			}
		})
	}
}
