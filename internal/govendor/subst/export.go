// Package subst is an excerpt from x/tools/go/ssa responsible for performing
// type substitution in types defined in terms of type parameters with provided
// type arguments.
package subst

import (
	"go/token"
	"go/types"
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
//   - The given context must be non-nil to cache types.
//   - The given function may be nil for any package level types.
//     It must be non-nil for any types nested within a generic function.
//   - Given type arguments should not contain any types in the type parameters.
//   - The internal implementation is not concurrency-safe.
func New(tc *types.Context, fn *types.Func, tParams *types.TypeParamList, tArgs []types.Type) *Subster {
	if tParams.Len() == 0 && len(tArgs) == 0 {
		return nil
	}

	if fn == nil {
		fn = types.NewFunc(token.NoPos, nil, "$substPseudoFunc",
			types.NewSignatureType(nil, nil, nil, nil, nil, false))
	}

	subst := makeSubster(tc, fn, tParams, tArgs, false)
	return &Subster{
		impl: subst,
	}
}

// Type returns a version of typ with all references to type parameters replaced
// with the corresponding type arguments.
func (s *Subster) Type(typ types.Type) types.Type {
	if s == nil {
		return typ
	}
	return s.impl.typ(typ)
}
