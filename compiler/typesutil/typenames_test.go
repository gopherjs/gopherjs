package typesutil

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/gopherjs/gopherjs/internal/srctesting"
)

func typeNameOpts() cmp.Options {
	return cmp.Options{
		cmp.Transformer("TypeName", func(name *types.TypeName) string {
			return types.ObjectString(name, nil)
		}),
	}
}

func TestTypeNames(t *testing.T) {
	src := `package test
	
	type A int
	type B int
	type C int
	`
	fset := token.NewFileSet()
	_, pkg := srctesting.Check(t, fset, srctesting.Parse(t, fset, src))
	A := srctesting.LookupObj(pkg, "A").(*types.TypeName)
	B := srctesting.LookupObj(pkg, "B").(*types.TypeName)
	C := srctesting.LookupObj(pkg, "C").(*types.TypeName)

	tn := TypeNames{}
	tn.Add(A)
	tn.Add(B)
	tn.Add(A)
	tn.Add(C)
	tn.Add(B)

	got := tn.Slice()
	want := []*types.TypeName{A, B, C}

	if diff := cmp.Diff(want, got, typeNameOpts()); diff != "" {
		t.Errorf("tn.Slice() returned diff (-want,+got):\n%s", diff)
	}
}
