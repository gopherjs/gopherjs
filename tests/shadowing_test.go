package tests

import "testing"

// These tests are based on https://github.com/gopherjs/gopherjs/issues/757
// and https://github.com/gopherjs/gopherjs/issues/1003.
//
// The code was failing because the JS would fail on the shadowed method, `Do`.
// Go says that a struct field name shadows a method being promoted from an
// embedded type. This test checks the several ways of shadowing of methods and
// preventing ambiguous selectors from being promoted.
//
// See https://go.dev/ref/spec#Selectors:
// > For a value x of type T or *T where T is not a pointer or interface type,
// > x.f denotes the field or method at the shallowest depth in T where there
// > is such an f. If there is not exactly one f with shallowest depth, the
// > selector expression is illegal.

type (
	doer            interface{ Do() string }
	doAnother       interface{ Do() string }
	doEmbedded      struct{}
	doEmbeddedAgain struct{}
	doValueEmbedded struct{}
)

func (*doEmbedded) Do() string      { return `Do` }
func (*doEmbeddedAgain) Do() string { return `Do it again` }
func (doValueEmbedded) Do() string  { return `Do value` }

// `container.Do` shadows `doEmbedded.Do`.
// This is based on https://github.com/gopherjs/gopherjs/issues/757
func Test_Shadow1(t *testing.T) {
	type container struct {
		doEmbedded
		Do string
	}

	c := &container{}
	shadowCheck(t, c, `Do not`)
	shadowCheck(t, c.doEmbedded, `Do not`)
	shadowCheck(t, &c.doEmbedded, `Do`)
}

// `dontainer.Do` shadows `doEmbedded.Do`.
func Test_Shadow2(t *testing.T) {
	type container struct {
		*doEmbedded
		Do string
	}
	c := &container{}
	shadowCheck(t, c, `Do not`)
	shadowCheck(t, c.doEmbedded, `Do`)
}

// `embedded.Do` is callable from `*container` but not `*container.doEmbedded`
// since `*container.Do` calls `Do` with `*doEmbedded` and `Do` can not be called
// with a non-pointer to embedded.
func Test_Shadow3(t *testing.T) {
	type container struct{ doEmbedded }

	c := &container{}
	shadowCheck(t, c, `Do`)
	shadowCheck(t, c.doEmbedded, `Do not`)
}

// This complements `Test_Shadow3` to check that `container.Do` can be called
// and `container.doEmbedded.Do` can also be called.
func Test_Shadow4(t *testing.T) {
	type container struct{ *doEmbedded }

	c := &container{}
	shadowCheck(t, c, `Do`)
	shadowCheck(t, c.doEmbedded, `Do`)
}

type Shadow5Container struct{ *doEmbedded }

func (e *Shadow5Container) Do() string { return `Try to do` }

// `Container5.Do` shadows `doEmbedded.Do`.
func Test_Shadow5(t *testing.T) {
	c := &Shadow5Container{}
	shadowCheck(t, c, `Try to do`)
	shadowCheck(t, c.doEmbedded, `Do`)
}

// `container.Do` shadows the interface's method `doer.Do`.
func Test_Shadow6(t *testing.T) {
	type container struct {
		doer
		Do string
	}

	c := &container{doer: &doEmbedded{}}
	shadowCheck(t, c, `Do not`)
	shadowCheck(t, c.doer, `Do`)
}

// `container.doEmbedded.Do` and `container.doEmbeddedAgain.Do` are ambiguous
// so `container.Do` can not be called.
func Test_Shadow7(t *testing.T) {
	type container struct {
		*doEmbedded
		*doEmbeddedAgain
	}

	c := &container{}
	shadowCheck(t, c, `Do not`)
	shadowCheck(t, c.doEmbedded, `Do`)
	shadowCheck(t, c.doEmbeddedAgain, `Do it again`)
}

// This is similar to `Test_Shadow7` but checks the ambiguity goes away when the methods
// are not in contention at the same level of embedding.
// `container.EmbedHolder.doEmbeddedAgain.Do` is deeper than `container.doEmbedded.Do`
// so `container.doEmbedded.Do` is called with `container.Do`, even through
// `container.EmbedHolder.Do` is also able to be called.
func Test_Shadow8(t *testing.T) {
	type (
		EmbedHolder struct{ *doEmbeddedAgain }
		container   struct {
			*doEmbedded
			EmbedHolder
		}
	)

	c := &container{}
	shadowCheck(t, c, `Do`)
	shadowCheck(t, c.doEmbedded, `Do`)
	shadowCheck(t, c.EmbedHolder, `Do it again`)
	shadowCheck(t, c.doEmbeddedAgain, `Do it again`)
}

// The field `Inner.Do` is ambiguous with the method `doEmbedded.Do`.
func Test_Shadow9(t *testing.T) {
	type (
		Inner struct {
			doEmbedded
			Do string
		}
		container struct {
			*doEmbedded
			*Inner
		}
	)

	c := &container{}
	shadowCheck(t, c, `Do not`)
	shadowCheck(t, c.doEmbedded, `Do`)
	shadowCheck(t, c.Inner, `Do not`)
}

// The method `Do` found in the field `Do` (e.g. `Do.doEmbedded.Do`) is ambiguous
// with embedded field `Do` itself.
func Test_Shadow10(t *testing.T) {
	type (
		Do        struct{ *doEmbedded }
		container struct{ *Do }
	)

	c := &container{Do: &Do{}}
	shadowCheck(t, c, `Do not`)
	shadowCheck(t, c.Do, `Do`)
}

// The ambiguity in `Inner` means that `Inner` does not have a `Do` method,
// meaning `container.Do` is not ambiguous and will call `container.doEmbedded.Do`.
func Test_Shadow11(t *testing.T) {
	type (
		Inner struct {
			*doEmbedded
			*doEmbeddedAgain
		}
		container struct {
			*doEmbedded
			*Inner
		}
	)

	c := &container{Inner: &Inner{}}
	shadowCheck(t, c, `Do`)
	shadowCheck(t, c.doEmbedded, `Do`)
	shadowCheck(t, c.Inner, `Do not`)
}

// `container.DoerHolder.doer` and `container.doEmbedded` are ambiguous even
// though `DoerHolder.doer` is deeper because embedded interfaces contribute
// methods to the embedding interface meaning `DoerHolder` contains a `Do` method
// thus `container.DoerHolder.Do` is at the same level as `container.doEmbedded.Do`.
func Test_Shadow12(t *testing.T) {
	type (
		DoerHolder interface{ doer }
		container  struct {
			DoerHolder
			*doEmbedded
		}
	)

	c := &container{DoerHolder: &doEmbedded{}}
	shadowCheck(t, c, `Do not`)
	shadowCheck(t, c.DoerHolder, `Do`)
}

// The field `Do` is a func type so it is callable via `container.Do` and
// will block `doEmbedded.Do` just like the string in `Test_Shadow1` does.
// However `container` does not duck-type to `doer` because `Do` is a field.
func Test_Shadow13(t *testing.T) {
	type container struct {
		*doEmbedded
		Do func() string
	}

	c := &container{Do: func() string { return `Don't` }}
	shadowCheck(t, c, `Do not`)
	shadowCheck(t, c.doEmbedded, `Do`)
}

// The embedded value must still promote the `Do` method implemented with a value receiver.
func Test_Shadow14(t *testing.T) {
	type container struct{ doValueEmbedded }

	c := &container{}
	shadowCheck(t, c, `Do value`)
	shadowCheck(t, c.doValueEmbedded, `Do value`)
}

// The `Do` field in `WithDoField` and the `Do` method in `doEmbedded`
// are at the same level so cause `Do` to be ambiguous in `container`.
// Even though `WithDoField.Do` can be called, since `Do` is a field `WithDoField`
// will not duck-type to `doer`.
func Test_Shadow15(t *testing.T) {
	type (
		WithDoField struct{ Do func() string }
		container   struct {
			*doEmbedded
			*WithDoField
		}
	)

	c := &container{
		WithDoField: &WithDoField{
			Do: func() string { return `Do not try` },
		},
	}
	shadowCheck(t, c, `Do not`)
	shadowCheck(t, c.doEmbedded, `Do`)
	shadowCheck(t, c.WithDoField, `Do not`)
}

// The two fields are both interfaces in contention because they both define
// `Do` methods meaning that `container.Do` is ambiguous.
// This is based off of https://github.com/gopherjs/gopherjs/issues/1003
func Test_Shadow16(t *testing.T) {
	type container struct {
		doer
		doAnother
	}

	c := &container{
		doer:      &doEmbedded{},
		doAnother: &doEmbedded{},
	}
	shadowCheck(t, c, `Do not`)
	shadowCheck(t, c.doer, `Do`)
	shadowCheck(t, c.doAnother, `Do`)
}

// shadowCheck does a runtime type check of a against `doer` to test the
// `$methodSet` method in the prelude.
func shadowCheck(t *testing.T, a any, want string) {
	t.Helper()
	got := `Do not`
	if aa, ok := a.(doer); ok {
		got = aa.Do()
	}
	if got != want {
		t.Errorf("expected %T to return %q but got %q\n", a, want, got)
	}
}
