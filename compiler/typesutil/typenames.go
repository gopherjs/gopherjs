package typesutil

import "go/types"

// TypeNames implements an ordered set of *types.TypeName pointers.
//
// The set is ordered to ensure deterministic behavior across compiler runs.
type TypeNames struct {
	known map[*types.TypeName]struct{}
	order []*types.TypeName
}

// Add a type name to the test. If the type name has been previously added,
// this operation is a no-op. Two type names are considered equal iff they have
// the same memory address.
func (tn *TypeNames) Add(name *types.TypeName) {
	if _, ok := tn.known[name]; ok {
		return
	}
	if tn.known == nil {
		tn.known = map[*types.TypeName]struct{}{}
	}
	tn.order = append(tn.order, name)
	tn.known[name] = struct{}{}
}

// Slice returns set elements in the order they were first added to the set.
func (tn *TypeNames) Slice() []*types.TypeName {
	return tn.order
}
