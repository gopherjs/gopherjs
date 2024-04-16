package typeparams

import (
	"errors"
	"testing"

	"github.com/gopherjs/gopherjs/internal/srctesting"
)

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
