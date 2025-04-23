// Package subst is an excerpt from x/tools/go/ssa responsible for performing
// type substitution in types defined in terms of type parameters with provided
// type arguments.
package subst

import (
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
func New(tc *types.Context, tParams []*types.TypeParam, tArgs []types.Type) *Subster {
	assert(len(tParams) == len(tArgs), "New() argument count must match")

	if len(tParams) == 0 {
		return nil
	}

	subst := &subster{
		replacements: make(map[*types.TypeParam]types.Type, len(tParams)),
		cache:        make(map[types.Type]types.Type),
		ctxt:         tc,
		scope:        nil,
		debug:        false,
	}
	for i := 0; i < len(tParams); i++ {
		subst.replacements[tParams[i]] = tArgs[i]
	}
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

func (s *Subster) String() string { // TODO(grantnelson-wf): remove
	if s == nil || s.impl == nil {
		return `<nil Subster>`
	}
	return s.impl.String()
}
