package typeparams

import (
	"errors"
	"go/token"
	"go/types"
	"testing"

	"github.com/gopherjs/gopherjs/internal/srctesting"
)

func TestHasTypeParams(t *testing.T) {
	pkg := types.NewPackage("test/pkg", "pkg")
	empty := types.NewInterfaceType(nil, nil)
	tParams := func() []*types.TypeParam {
		return []*types.TypeParam{
			types.NewTypeParam(types.NewTypeName(token.NoPos, pkg, "T", types.Typ[types.String]), empty),
		}
	}

	tests := []struct {
		descr string
		typ   types.Type
		want  bool
	}{{
		descr: "generic function",
		typ:   types.NewSignatureType(nil, nil, tParams(), nil, nil, false),
		want:  true,
	}, {
		descr: "generic method",
		typ:   types.NewSignatureType(types.NewVar(token.NoPos, pkg, "t", nil), tParams(), nil, nil, nil, false),
		want:  true,
	}, {
		descr: "regular function",
		typ:   types.NewSignatureType(nil, nil, nil, nil, nil, false),
		want:  false,
	}, {
		descr: "generic type",
		typ: func() types.Type {
			typ := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Typ", nil), types.Typ[types.String], nil)
			typ.SetTypeParams(tParams())
			return typ
		}(),
		want: true,
	}, {
		descr: "regular named type",
		typ:   types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Typ", nil), types.Typ[types.String], nil),
		want:  false,
	}, {
		descr: "built-in type",
		typ:   types.Typ[types.String],
		want:  false,
	}}

	for _, test := range tests {
		t.Run(test.descr, func(t *testing.T) {
			got := HasTypeParams(test.typ)
			if got != test.want {
				t.Errorf("Got: HasTypeParams(%v) = %v. Want: %v.", test.typ, got, test.want)
			}
		})
	}
}

func TestRequiresGenericsSupport(t *testing.T) {
	t.Run("generic func", func(t *testing.T) {
		f := srctesting.New(t)
		src := `package foo
		func foo[T any](t T) {}`
		info, _ := f.Check("pkg/foo", f.Parse("foo.go", src))

		err := RequiresGenericsSupport(info)
		if !errors.Is(err, errDefinesGenerics) {
			t.Errorf("Got: RequiresGenericsSupport() = %v. Want: %v", err, errDefinesGenerics)
		}
	})

	t.Run("generic type", func(t *testing.T) {
		f := srctesting.New(t)
		src := `package foo
		type Foo[T any] struct{t T}`
		info, _ := f.Check("pkg/foo", f.Parse("foo.go", src))

		err := RequiresGenericsSupport(info)
		if !errors.Is(err, errDefinesGenerics) {
			t.Errorf("Got: RequiresGenericsSupport() = %v. Want: %v", err, errDefinesGenerics)
		}
	})

	t.Run("imported generic instance", func(t *testing.T) {
		f := srctesting.New(t)
		f.Info = nil // Do not combine type checking info from different packages.
		src1 := `package foo
		type Foo[T any] struct{t T}`
		f.Check("pkg/foo", f.Parse("foo.go", src1))

		src2 := `package bar
		import "pkg/foo"
		func bar() { _ = foo.Foo[int]{} }`
		info, _ := f.Check("pkg/bar", f.Parse("bar.go", src2))

		err := RequiresGenericsSupport(info)
		if !errors.Is(err, errInstantiatesGenerics) {
			t.Errorf("Got: RequiresGenericsSupport() = %v. Want: %v", err, errInstantiatesGenerics)
		}
	})

	t.Run("no generic usage", func(t *testing.T) {
		f := srctesting.New(t)
		src := `package foo
		type Foo struct{}
		func foo() { _ = Foo{} }`
		info, _ := f.Check("pkg/foo", f.Parse("foo.go", src))

		err := RequiresGenericsSupport(info)
		if err != nil {
			t.Errorf("Got: RequiresGenericsSupport() = %v. Want: nil", err)
		}
	})
}
