package typeparams

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"testing"

	"github.com/google/go-cmp/cmp"
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

func Test_FindNestingFunc(t *testing.T) {
	src := `package main

		type bob int
		func (a bob) riker() any {
			type bill struct{ b int }
			return bill{b: int(a)}
		}

		type milo[T any] struct{}
		func (c *milo[U]) mario() any {
			type homer struct{ d U }
			return homer{}
		}

		func bart() any {
			e := []milo[int]{{}}
			f := &e[0]
			return f.mario()
		}

		func jack() any {
			type linus interface{
				interface {
					marvin()
				}
				luke()
			}
			type owen interface {
				linus
				isaac()
			}
			return owen(nil)
		}

		func bender() any {
			charles := func() any {
				type arthur struct{ h int }
				return arthur{h: 42}
			}
			return charles()
		}
		
		var ned = func() any {
			type elmer struct{ i int }
			return elmer{i: 42}
		}()

		func garfield(count int) {
			calvin:
			for j := 0; j < count; j++ {
				if j == 5 {
					break calvin
				}
			}
		}`

	f := srctesting.New(t)
	file := f.Parse("main.go", src)
	info, _ := f.Check("test", file)

	// Collect all objects and find nesting functions.
	// The results will be ordered by position in the file.
	results := []string{}
	ast.Inspect(file, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			obj := info.ObjectOf(id)
			if _, isVar := obj.(*types.Var); isVar {
				// Skip variables, some variables (e.g. receivers) are not inside
				// a function's scope in go1.19 but in later versions they are.
				return true
			}
			if named, isNamed := obj.(*types.TypeName); isNamed {
				if _, isTP := named.Type().(*types.TypeParam); isTP {
					// Skip type parameters since they are not inside
					// a function's scope in go1.19 but in later versions they are.
					return true
				}
			}

			fn := FindNestingFunc(obj)
			fnName := ``
			if fn != nil {
				fnName = fn.FullName()
			}
			results = append(results, fmt.Sprintf("%3d) %s => %s", id.Pos(), id.Name, fnName))
		}
		return true
	})

	diff := cmp.Diff([]string{
		// package main (nil object)
		`  9) main => `,

		// type bob int
		` 22) bob => `,
		` 26) int => `, // use of basic

		// func (a bob) riker() any { ... }
		` 40) bob => `,
		` 45) riker => `,
		` 53) any => `,
		` 67) bill => (test.bob).riker`, // def
		` 82) int => `,
		` 98) bill => (test.bob).riker`, // use
		`106) int => `,

		// type milo[T any] struct {}
		`126) milo => `,
		`133) any => `,

		// func (c *milo[U]) mario() any { ... }
		`158) milo => `,
		`167) mario => `,
		`175) any => `,
		`189) homer => (*test.milo[U]).mario`, // def
		`219) homer => (*test.milo[U]).mario`, // use

		// func bart() any { ... }
		`239) bart => `,
		`246) any => `,
		`262) milo => `, // use of non-local defined type
		`267) int => `,
		`302) mario => `, // use of method on non-local defined type

		// func jack() any { ... }
		`322) jack => `,
		`329) any => `,
		`343) linus => test.jack`, // def
		`381) marvin => `,         // method def
		`400) luke => `,           // method def
		`420) owen => test.jack`,  // def
		`441) linus => test.jack`, // use
		`451) isaac => `,          // method def
		`474) owen => test.jack`,  // use
		`479) nil => `,            // use of nil

		// func bender() any { ... }
		`496) bender => `,
		`505) any => `,
		`532) any => `,
		`547) arthur => test.bender`, // def inside func lit
		`564) int => `,
		`581) arthur => test.bender`, // use

		// var ned = func() any { ... }
		`646) any => `,
		`660) elmer => `, // def inside package-level func lit
		`676) int => `,
		`692) elmer => `, // use

		// func garfield(count int) { ... }
		`719) garfield => `,
		`734) int => `,
		`744) calvin => `, // local label def
		`811) calvin => `, // label break
	}, results)
	if len(diff) > 0 {
		t.Errorf("FindNestingFunc() mismatch (-want +got):\n%s", diff)
	}
}
