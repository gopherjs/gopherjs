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
	// concrete type or is a method or function declaration.
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

	switch inst.Object.(type) {
	case *types.Builtin, *types.Func:
		// Nothing can depend on a function so we don't need to set a type.
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
		// The signature parameters and results are not added as dependencies
		// because they are not used in initialization.

	case *types.Map:
		i.addDep(t.Key())
		i.addDep(t.Elem())

	case interface{ Elem() types.Type }:
		// Handles *types.Pointer, *types.Slice, *types.Array, and *types.Chan
		i.addDep(t.Elem())
	}
}

func (i *Info) skipDep(t types.Type) bool {
	if t == nil {
		return true // skip nil types
	}
	if typesutil.IsJsObject(t) {
		return true // skip *js.Object
	}

	switch t := t.(type) {
	case *types.Basic:
		return true // skip basic types like `int`, `string`, `unsafe.Pointer` etc.

	case *types.Named:
		if t.Obj() == nil || t.Obj().Pkg() == nil {
			return true // skip objects in universal scope, e.g. `error`
		}

	case *types.Struct:
		if t.NumFields() == 0 {
			return true // skip `struct{}`
		}

	case *types.Interface:
		if t.Empty() {
			return true // skip `any`
		}

	case *types.Pointer:
		if tn, ok := t.Elem().(*types.Named); ok && tn.Obj() != nil && tn.Obj().Pkg() != nil &&
			tn.Obj().Pkg().Path() == "internal/reflectlite" && tn.Obj().Name() == "rtype" {
			return true // skip `*internal/reflectlite.rtype`
		}
	}
	return false
}

func (i *Info) addDep(t types.Type) {
	if i.skipDep(t) {
		return
	}
	if i.dep == nil {
		i.dep = make(map[types.Type]struct{})
	}
	i.dep[t] = struct{}{}
}
