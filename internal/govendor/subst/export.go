// Package subst is an excerpt from x/tools/go/ssa responsible for performing
// type substitution in types defined in terms of type parameters with provided
// type arguments.
package subst

import (
	"fmt"
	"go/types"
	"sort"
	"strings"
)

// To simplify future updates of the borrowed code, we minimize modifications
// to it as much as possible. This file implements an exported interface to the
// original code for us to use.

// Subster performs type parameter substitution.
type Subster struct {
	nest *Subster
	impl *subster
}

// New creates a new Subster with a given list of type parameters and matching args.
//
//   - This may return a nil if there is no substitution to be done.
//     Using a nil Subster will always return the original type.
//   - Given type arguments should not contain any types in the type parameters.
//   - The internal implementation is not concurrency-safe.
//   - If a non-nil nest is given, this subster will use that nest to
//     perform substitution as well to allow for nested types.
func New(tc *types.Context, tParams *types.TypeParamList, tArgs []types.Type, nest *Subster) *Subster {
	tpLen := 0
	if tParams != nil {
		tpLen = tParams.Len()
	}

	assert(tpLen == len(tArgs), "New() argument count must match")

	if tpLen == 0 && len(tArgs) == 0 {
		return nest
	}

	subst := makeSubster(tc, nil, tParams, tArgs, false)
	return &Subster{nest: nest, impl: subst}
}

// Type returns a version of typ with all references to type parameters replaced
// with the corresponding type arguments.
func (s *Subster) Type(typ types.Type) types.Type {
	if s == nil {
		return typ
	}
	typ = s.impl.typ(typ)
	if s.nest != nil {
		typ = s.nest.Type(typ)
	}
	return typ
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
	nestStr := ``
	if s.nest != nil {
		nestStr = s.nest.String() + `:`
	}
	return nestStr + `{` + strings.Join(parts, `, `) + `}`
}
