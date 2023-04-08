package typesutil

import (
	"go/ast"
	"go/token"
	"go/types"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/gopherjs/gopherjs/internal/srctesting"
)

func TestAnonymousTypes(t *testing.T) {
	t1 := types.NewSlice(types.Typ[types.String])
	t1Name := types.NewTypeName(token.NoPos, nil, "sliceType$1", t1)

	t2 := types.NewMap(types.Typ[types.Int], t1)
	t2Name := types.NewTypeName(token.NoPos, nil, "mapType$1", t2)

	typs := []struct {
		typ  types.Type
		name *types.TypeName
	}{
		{typ: t1, name: t1Name},
		{typ: t2, name: t2Name},
	}

	anonTypes := AnonymousTypes{}
	for _, typ := range typs {
		anonTypes.Register(typ.name, typ.typ)
	}

	for _, typ := range typs {
		t.Run(typ.name.Name(), func(t *testing.T) {
			got := anonTypes.Get(typ.typ)
			if got != typ.name {
				t.Errorf("Got: anonTypes.Get(%v) = %v. Want: %v.", typ.typ, typ.name, got)
			}
		})
	}

	gotNames := []string{}
	for _, name := range anonTypes.Ordered() {
		gotNames = append(gotNames, name.Name())
	}
	wantNames := []string{"sliceType$1", "mapType$1"}
	if !cmp.Equal(wantNames, gotNames) {
		t.Errorf("Got: anonTypes.Ordered() = %v. Want: %v (in the order of registration)", gotNames, wantNames)
	}
}

func TestIsGeneric(t *testing.T) {
	T := types.NewTypeParam(types.NewTypeName(token.NoPos, nil, "T", nil), types.NewInterface(nil, nil))

	tests := []struct {
		typ  types.Type
		want bool
	}{
		{
			typ:  T,
			want: true,
		}, {
			typ:  types.Typ[types.Int],
			want: false,
		}, {
			typ:  types.NewArray(types.Typ[types.Int], 1),
			want: false,
		}, {
			typ:  types.NewArray(T, 1),
			want: true,
		}, {
			typ:  types.NewChan(types.SendRecv, types.Typ[types.Int]),
			want: false,
		}, {
			typ:  types.NewChan(types.SendRecv, T),
			want: true,
		}, {
			typ: types.NewInterfaceType(
				[]*types.Func{
					types.NewFunc(token.NoPos, nil, "X", types.NewSignatureType(
						nil, nil, nil, types.NewTuple(types.NewVar(token.NoPos, nil, "x", types.Typ[types.Int])), nil, false,
					)),
				},
				[]types.Type{
					types.NewNamed(types.NewTypeName(token.NoPos, nil, "myInt", nil), types.Typ[types.Int], nil),
				},
			),
			want: false,
		}, {
			typ: types.NewInterfaceType(
				[]*types.Func{
					types.NewFunc(token.NoPos, nil, "X", types.NewSignatureType(
						nil, nil, nil, types.NewTuple(types.NewVar(token.NoPos, nil, "x", T)), nil, false,
					)),
				},
				[]types.Type{
					types.NewNamed(types.NewTypeName(token.NoPos, nil, "myInt", nil), types.Typ[types.Int], nil),
				},
			),
			want: true,
		}, {
			typ:  types.NewMap(types.Typ[types.Int], types.Typ[types.String]),
			want: false,
		}, {
			typ:  types.NewMap(T, types.Typ[types.String]),
			want: true,
		}, {
			typ:  types.NewMap(types.Typ[types.Int], T),
			want: true,
		}, {
			typ:  types.NewNamed(types.NewTypeName(token.NoPos, nil, "myInt", nil), types.Typ[types.Int], nil),
			want: false,
		}, {
			typ:  types.NewPointer(types.Typ[types.Int]),
			want: false,
		}, {
			typ:  types.NewPointer(T),
			want: true,
		}, {
			typ:  types.NewSlice(types.Typ[types.Int]),
			want: false,
		}, {
			typ:  types.NewSlice(T),
			want: true,
		}, {
			typ: types.NewSignatureType(
				nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, nil, "x", types.Typ[types.Int])),   // params
				types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])), // results
				false,
			),
			want: false,
		}, {
			typ: types.NewSignatureType(
				nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, nil, "x", T)),                      // params
				types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])), // results
				false,
			),
			want: true,
		}, {
			typ: types.NewSignatureType(
				nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, nil, "x", types.Typ[types.Int])), // params
				types.NewTuple(types.NewVar(token.NoPos, nil, "", T)),                     // results
				false,
			),
			want: true,
		}, {
			typ: types.NewStruct([]*types.Var{
				types.NewVar(token.NoPos, nil, "x", types.Typ[types.Int]),
			}, nil),
			want: false,
		}, {
			typ: types.NewStruct([]*types.Var{
				types.NewVar(token.NoPos, nil, "x", T),
			}, nil),
			want: true,
		},
	}

	for _, test := range tests {
		t.Run(test.typ.String(), func(t *testing.T) {
			got := IsGeneric(test.typ)
			if got != test.want {
				t.Errorf("Got: IsGeneric(%v) = %v. Want: %v.", test.typ, got, test.want)
			}
		})
	}
}

func TestCanonicalTypeParamMap(t *testing.T) {
	src := `package main
	type A[T any] struct{}
	func (a A[T1]) Method(t T1) {}

	func Func[U any](u U) {}
	`
	fset := token.NewFileSet()
	f := srctesting.Parse(t, fset, src)
	info, _ := srctesting.Check(t, fset, f)

	// Extract relevant information about the method Method.
	methodDecl := f.Decls[1].(*ast.FuncDecl)
	if methodDecl.Name.String() != "Method" {
		t.Fatalf("Unexpected function at f.Decls[2] position: %q. Want: Method.", methodDecl.Name.String())
	}
	method := info.Defs[methodDecl.Name]
	T1 := method.Type().(*types.Signature).Params().At(0).Type().(*types.TypeParam)
	if T1.Obj().Name() != "T1" {
		t.Fatalf("Unexpected type of the Func's first argument: %s. Want: T1.", T1.Obj().Name())
	}

	// Extract relevant information about the standalone function Func.
	funcDecl := f.Decls[2].(*ast.FuncDecl)
	if funcDecl.Name.String() != "Func" {
		t.Fatalf("Unexpected function at f.Decls[2] position: %q. Want: Func.", funcDecl.Name.String())
	}
	fun := info.Defs[funcDecl.Name]
	U := fun.Type().(*types.Signature).Params().At(0).Type().(*types.TypeParam)
	if U.Obj().Name() != "U" {
		t.Fatalf("Unexpected type of the Func's first argument: %s. Want: U.", U.Obj().Name())
	}

	cm := NewCanonicalTypeParamMap([]*ast.FuncDecl{methodDecl, funcDecl}, info)

	// Method's type params canonicalized to their receiver type's.
	got := cm.Lookup(T1)
	if got.Obj().Name() != "T" {
		t.Errorf("Got canonical type parameter %q for %q. Want: T.", got, T1)
	}

	// Function's type params don't need canonicalization.
	got = cm.Lookup(U)
	if got.Obj().Name() != "U" {
		t.Errorf("Got canonical type parameter %q for %q. Want: U.", got, U)
	}
}
