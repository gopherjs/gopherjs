// Package subst is an excerpt from x/tools/go/ssa responsible for performing
// type substitution in types defined in terms of type parameters with provided
// type arguments.
package subst

import (
	"fmt"
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
func New(tc *types.Context, tParams *types.TypeParamList, tArgs []types.Type) *Subster {
	if tParams.Len() != len(tArgs) {
		panic(fmt.Errorf(`New() argument count must match (%d != %d)`, tParams.Len(), len(tArgs)))
	}

	if tParams.Len() == 0 && len(tArgs) == 0 {
		return nil
	}

	subst := makeSubster(tc, nil, tParams, tArgs, false)
	return &Subster{impl: subst}
}

// Type returns a version of typ with all references to type parameters replaced
// with the corresponding type arguments.
func (s *Subster) Type(typ types.Type) types.Type {
	if s == nil {
		return typ
	}
	return s.impl.typ(typ)
}
