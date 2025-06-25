package grouper

import (
	"go/types"

	"github.com/gopherjs/gopherjs/compiler/internal/typeparams"
	"github.com/gopherjs/gopherjs/compiler/typesutil"
)

type Info struct {
	// Group is the group number for initializing this declaration.
	// Since parameter types and field types aren't taken into account when
	// ordering the groups, the declarations in the same group should still
	// be initialized in the same order as they were declared based on imports.
	Group int

	// typ is the concrete type this declaration is associated with.
	// This may be nil for declarations that do not have an associated
	// concrete type.
	typ types.Type

	// dep is a set of types that this declaration depends on.
	// This may be empty if there are no dependencies.
	dep map[types.Type]struct{}
}

// SetInstance sets the types and dependencies used by the grouper to represent
// the declaration this grouper info is attached to.
func (i *Info) SetInstance(tc *types.Context, inst typeparams.Instance) {
	i.setType(tc, inst)
	i.setAllDeps(tc, inst)
}

func (i *Info) setType(tc *types.Context, inst typeparams.Instance) {
	if inst.Object == nil {
		return
	}

	switch t := inst.Object.Type().(type) {
	case *types.Named:
		i.typ = inst.Resolve(tc)
	default:
		i.typ = t
	}
}

func (i *Info) setAllDeps(tc *types.Context, inst typeparams.Instance) {
	for _, nestArg := range inst.TNest {
		i.addDep(nestArg)
	}
	for _, tArg := range inst.TArgs {
		i.addDep(tArg)
	}

	switch t := inst.Object.Type().(type) {
	case interface{ TypeArgs() *types.TypeList }:
		// Handles *type.Named and *types.Alias (in go1.22)
		for j := t.TypeArgs().Len() - 1; j >= 0; j-- {
			i.addDep(t.TypeArgs().At(j))
		}

	case *types.Signature:
		if recv := typesutil.RecvType(t); recv != nil {
			recvInst := typeparams.Instance{
				Object: recv.Obj(),
				TNest:  inst.TNest,
				TArgs:  inst.TArgs,
			}
			i.addDep(recvInst.Resolve(tc))
		}

	case *types.Map:
		i.addDep(t.Key())
		i.addDep(t.Elem())

	case interface{ Elem() types.Type }:
		// Handles *types.Pointer, *types.Slice, *types.Array, and *types.Chan
		i.addDep(t.Elem())
	}
}

func (i *Info) addDep(t types.Type) {
	switch t.(type) {
	case nil, *types.Basic:
		// Nil and Basic types aren't used as dependencies
		// since they don't have unique declarations.
		return
	}

	if i.dep == nil {
		i.dep = make(map[types.Type]struct{})
	}
	i.dep[t] = struct{}{}
}
