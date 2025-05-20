// Package subst is an excerpt from x/tools/go/ssa responsible for performing
// type substitution in types defined in terms of type parameters with provided
// type arguments.
package subst

import "go/types"

// To simplify future updates of the borrowed code, we minimize modifications
// to it as much as possible. This file implements an exported interface to the
// original code for us to use.

// Subster performs type parameter substitution.
type Subster struct {
	impl *subster
}

// New creates a new Subster with a given a map from type parameters and the arguments
// that should be used to replace them. If the map is empty, nil is returned.
// The function `fn` is used to determine the nesting context of the substitution.
func New(tc *types.Context, fn *types.Func, replacements map[*types.TypeParam]types.Type) *Subster {
	if len(replacements) == 0 {
		return nil
	}

	subst := makeSubster(tc, fn, nil, nil, false)
	subst.replacements = replacements
	return &Subster{impl: subst}
}

// Type returns a version of typ with all references to type parameters
// replaced with the corresponding type arguments.
func (s *Subster) Type(typ types.Type) types.Type {
	if s == nil {
		return typ
	}
	return s.impl.typ(typ)
}

// Types returns a version of ts with all references to type parameters
// replaced with the corresponding type arguments.
func (s *Subster) Types(ts []types.Type) []types.Type {
	if s == nil {
		return ts
	}
	return s.impl.types(ts)
}
