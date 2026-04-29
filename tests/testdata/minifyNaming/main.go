// This reproduces a minified-name collision triggered by generics
// and causes a JS exception like: `TypeError: b.Inc is not a function`
//
// This is a simplification of a bug found by the `TestStructSorts`
// in the `slices` package.
//
// The issue was that when determining minified identifiers:
// The first instantiation of a method has a type pointer added to it `allVars`.
// The second instantiation hit the cached type pointer so did not add
// it to `allVars` causing the next local variable to use the same identifier
// as that type pointer. The duplicate identifiers can then clobber eachother
// that can lead to trying to call a method for the type pointer on a local
// variable without that method.

package main

type rng int64

func (r *rng) Inc() { *r++ }

// For the first instantiation `r`, `&r`, and `extra` get minified to `a`, `b`,
// and `c` respectively, but for the second instantiation, `extra` gets minified
// to `b` causing a conflict between `&r` and `extra`.
//
//go:noinline
func shape[T any](_ T) {
	r := rng(0)
	r.Inc()
	extra := 42
	_ = extra
	r.Inc() // In second instantiation this fails with `int(42).Inc()`.
}

func main() {
	shape[int](0)
	shape[string](``)
}
