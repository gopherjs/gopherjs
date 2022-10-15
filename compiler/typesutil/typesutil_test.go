package typesutil

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/google/go-cmp/cmp"
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
