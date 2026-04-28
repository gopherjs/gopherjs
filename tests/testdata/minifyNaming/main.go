// This package tests an issue found in slices/zsortanyfunc.go:breakPatternsCmpFunc
// (a generic function that becomes T[132], T[140], etc. in the minified output).
//
// The issue: varPtrName allocates and caches a "$ptr" alias name in the
// package-level pkgCtx.varPtrNames keyed by the *types.Var. For generic
// functions, the same *types.Var (the local variable inside the generic body)
// is shared by every instantiation. The first instantiation calls
// newVariable() which registers the chosen minified letter in its
// funcContext.allVars. Subsequent instantiations hit the cache and only append
// the cached name to fc.localVars (see utils.go:506-508), but they do NOT
// register it in fc.allVars. As a result, a later newLocalVariable() call in
// that funcContext can be assigned the same minified letter, producing a
// duplicate in the emitted "var ..." list (e.g. "var a, b, ..., j, j, k, ...").
//
// To trigger the bug we need:
//  1. A generic function with a local variable whose address is taken (so
//     varPtrName is invoked for it). A pointer-receiver method call on a
//     local var of an interface-constrained type works.
//  2. The same generic function instantiated with at least two different type
//     arguments, so the second instantiation hits the cached varPtrName
//     without re-registering it in allVars.
//  3. Enough additional locals (declared after the &-take site) that the
//     lowercase counter rolls up to the cached letter, producing the visible
//     duplicate-name collision.
package main

// rng has a pointer-receiver method, so calling rng.Next() on a local variable
// of type *rng implicitly takes its address and forces varPtrName allocation
// for that local var inside the generic function below.
type rng uint64

func (r *rng) Next() uint64 {
	*r = (*r)*6364136223846793005 + 1442695040888963407
	return uint64(*r)
}

// shuffleSeed is generic so it gets instantiated once per type argument. The
// local var `r` (an *rng-style state) has its address taken via r.Next().
// That allocates a "$ptr" minified name on the first instantiation; the second
// instantiation reuses the cached name without registering it in allVars.
//
//go:noinline
func shuffleSeed[T any](items []T, seed uint64) T {
	// Local var whose address is implicitly taken by the pointer-receiver
	// method call below. This is what triggers varPtrName().
	r := rng(seed)

	// First reference to &r — on the first instantiation this allocates a
	// minified "$ptr" letter (e.g. "j") and registers it in this
	// funcContext's allVars. On the second instantiation, varPtrName returns
	// the cached letter and only appends it to localVars, NOT allVars.
	_ = r.Next()

	// Now declare many more local vars. On the second instantiation, since
	// the cached "$ptr" letter is missing from allVars, newLocalVariable
	// will hand out the same letter again, producing a duplicate in the
	// emitted "var ..." list.
	a1 := r.Next()
	a2 := a1 + 1
	a3 := a2 + 1
	a4 := a3 + 1
	a5 := a4 + 1
	a6 := a5 + 1
	a7 := a6 + 1
	a8 := a7 + 1
	a9 := a8 + 1
	a10 := a9 + 1
	a11 := a10 + 1
	a12 := a11 + 1
	a13 := a12 + 1

	idx := int((a1 ^ a2 ^ a3 ^ a4 ^ a5 ^ a6 ^ a7 ^ a8 ^ a9 ^ a10 ^ a11 ^ a12 ^ a13) % uint64(len(items)))
	return items[idx]
}

// TODO: Need to rework this so that the duplicate "d" value is hit within a loop so that
// the problem actually causes an exception to be thrown, not just a quiet potential issue.
func main() {
	// Two distinct instantiations of the same generic function. The second
	// one is what produces the duplicate-name collision.
	ints := []int{1, 2, 3, 4, 5, 6, 7, 8}
	strs := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	println("int :", shuffleSeed(ints, 42))
	println("str :", shuffleSeed(strs, 42))
}
