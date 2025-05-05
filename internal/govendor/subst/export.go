// Package subst is an excerpt from x/tools/go/ssa responsible for performing
// type substitution in types defined in terms of type parameters with provided
// type arguments.
package subst

import (
	"fmt"
	"go/token"
	"go/types"
	"sort"
	"strings"
)

// To simplify future updates of the borrowed code, we minimize modifications
// to it as much as possible. This file implements an exported interface to the
// original code for us to use.

// Subster performs type parameter substitution.
type Subster struct {
	impl *subster
}

// New creates a new Subster with a given list of type parameters and matching args.
//
//   - This may return a nil if there is no substitution to be done.
//     Using a nil Subster will always return the original type.
//   - Given type arguments should not contain any types in the type parameters.
//   - The internal implementation is not concurrency-safe.
func New(tc *types.Context, tParams *types.TypeParamList, tArgs []types.Type) *Subster {
	return NewNested(tc, nil, nil, tParams, tArgs)
}

// NewNested creates a new Subster with a given list of type parameters and
// matching args for types nested within a function.
//
//   - This may return a nil if there is no substitution to be done.
//     Using a nil Subster will always return the original type.
//   - The given context must be non-nil to cache types.
//   - The given function may be nil for any package level types.
//     It must be non-nil for any types nested within a function.
//   - The given function type arguments will instantiate the function
//     and be used on any nested types.
//   - Given type arguments should not contain any types in the type parameters.
//   - The internal implementation is not concurrency-safe.
func NewNested(tc *types.Context, fn *types.Func, fnArgs []types.Type, tParams *types.TypeParamList, tArgs []types.Type) *Subster {
	assert(safeLen(tParams) == len(tArgs), "New() argument count must match")

	fnParams := getFuncTypeParams(fn)
	assert(fnParams.Len() == len(fnArgs), "New() function argument count must match")

	if safeLen(tParams) == 0 && len(tArgs) == 0 && safeLen(fnParams) == 0 && len(fnArgs) == 0 {
		return nil
	}

	if fn == nil {
		fn = types.NewFunc(token.NoPos, nil, "$substPseudoFunc",
			types.NewSignatureType(nil, nil, nil, nil, nil, false))
	}

	subst := makeSubster(tc, fn, tParams, tArgs, false)
	for i := 0; i < fnParams.Len(); i++ {
		subst.replacements[fnParams.At(i)] = fnArgs[i]
	}
	return &Subster{impl: subst}
}

func safeLen(tp interface{ Len() int }) int {
	if tp == nil {
		return 0
	}
	return tp.Len()
}

// getFuncTypeParams gets the type parameters of the given function.
// It will return either the receiver type parameters,
// the function type parameters, or nil if none found.
func getFuncTypeParams(fn *types.Func) *types.TypeParamList {
	if fn == nil {
		return nil
	}
	sig := fn.Type().(*types.Signature)
	if sig == nil {
		return nil
	}
	if tps := sig.RecvTypeParams(); tps != nil && tps.Len() > 0 {
		return tps
	}
	if tps := sig.TypeParams(); tps != nil && tps.Len() > 0 {
		return tps
	}
	return nil
}

// Type returns a version of typ with all references to type parameters replaced
// with the corresponding type arguments.
func (s *Subster) Type(typ types.Type) types.Type {
	if s == nil {
		return typ
	}
	return s.impl.typ(typ)
}

// String gets a strings representation of the replacement for debugging.
// The parameters are sorted by name not by the order in the type parameter list.
func (s *Subster) String() string {
	if s == nil || s.impl == nil {
		return `{}`
	}

	parts := make([]string, 0, len(s.impl.replacements))
	for tp, ta := range s.impl.replacements {
		parts = append(parts, fmt.Sprintf("%s->%s", tp, ta))
	}
	sort.Strings(parts)
	return `{` + strings.Join(parts, `, `) + `}`
}
