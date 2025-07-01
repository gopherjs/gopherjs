package grouper

import (
	"go/types"

	"github.com/gopherjs/gopherjs/compiler/internal/typeparams"
	"github.com/gopherjs/gopherjs/compiler/typesutil"
)

type Info struct {
	// Group is the group number for initializing this declaration.
	// The declarations in the same group should still be initialized in the
	// same order as they were declared based on imports first.
	Group int

	// name is the concrete named type that this declaration is associated with.
	// This may be nil for declarations that do not have an associated
	// concrete named type, such as functions and methods.
	name *types.Named

	// dep is a set of named types from other packages that this declaration
	// depends on. This may be empty if there are no dependencies.
	dep map[*types.Named]struct{}
}

// SetInstance sets the types and dependencies used by the grouper to represent
// the declaration this grouper info is attached to.
func (i *Info) SetInstance(tc *types.Context, inst typeparams.Instance) {
	i.setType(tc, inst)
	i.setAllDeps(tc, inst, true)
}

func (i *Info) setType(tc *types.Context, inst typeparams.Instance) {
	if inst.Object == nil {
		return
	}
	switch inst.Object.Type().(type) {
	// TODO(grantnelson-wf): Determine how to handle *types.Alias in go1.22
	case *types.Named:
		i.name = inst.Resolve(tc).(*types.Named)
	}
}

func (i *Info) setAllDeps(tc *types.Context, inst typeparams.Instance, skipSamePkg bool) {
	var pending []types.Type
	pending = append(pending, inst.TNest...)
	pending = append(pending, inst.TArgs...)

	switch {
	case inst.Object == nil:
		// shouldn't happen, but if it does, just check the type args.

	case i.name != nil:
		// If `i.name`` is set then we know we have a named type
		// that we have to dig into to find its dependencies.
		// By using `i.name` we know that the type has been resolved.
		tArgs := i.name.TypeArgs()
		for j := tArgs.Len() - 1; j >= 0; j-- {
			pending = append(pending, tArgs.At(j))
		}
		pending = append(pending, i.name.Underlying())

	case typesutil.IsMethod(inst.Object):
		// If the instance is a method, we need to resolve the receiver type
		// and the signature of the method to find its dependencies.
		sig := inst.Object.Type().(*types.Signature)
		if recv := typesutil.RecvType(sig); recv != nil {
			recvInst := typeparams.Instance{
				Object: recv.Obj(),
				TNest:  inst.TNest,
				TArgs:  inst.TArgs,
			}
			pending = append(pending, recvInst.Resolve(tc))
		}
		pending = append(pending, sig)

	default:
		// If `i.name` is not set and it isn't a method, we can add the type
		// as a dependency directly without needing to resolve it further.
		// This will take a type like `[]Cat` and add `Cat` as a dependency.
		pending = append(pending, inst.Object.Type())
	}

	i.seekDeps(pending, skipSamePkg)
}

func (i *Info) seekDeps(pending []types.Type, skipSamePkg bool) {
	touched := make(map[types.Type]struct{})
	for len(pending) > 0 {
		max := len(pending) - 1
		t := pending[max]
		pending = pending[:max]
		if _, ok := touched[t]; ok {
			continue // already processed this type
		}
		touched[t] = struct{}{}

		switch t := t.(type) {
		case *types.Basic:
			// ignore basic types like int, string, unsafe.Pointer, etc.

		case *types.Named:
			i.addDep(t, skipSamePkg)

		case *types.Struct:
			for j := t.NumFields() - 1; j >= 0; j-- {
				pending = append(pending, t.Field(j).Type())
			}

		case *types.Signature:
			for j := t.Params().Len() - 1; j >= 0; j-- {
				pending = append(pending, t.Params().At(j).Type())
			}
			for j := t.Results().Len() - 1; j >= 0; j-- {
				pending = append(pending, t.Results().At(j).Type())
			}

		case *types.Map:
			pending = append(pending, t.Key())
			pending = append(pending, t.Elem())

		case interface{ Elem() types.Type }:
			// Handles *types.Pointer, *types.Slice, *types.Array, and *types.Chan
			pending = append(pending, t.Elem())
		}
	}
}

func (i *Info) addDep(t *types.Named, skipSamePkg bool) {
	if t.Obj() == nil || t.Obj().Pkg() == nil {
		return // skip objects in universal scope, e.g. `error`
	}
	if typesutil.IsJsPackage(t.Obj().Pkg()) && t.Obj().Name() == "Object" {
		return // skip *js.Object
	}
	if skipSamePkg && i.name != nil && i.name.Obj() != nil &&
		i.name.Obj().Pkg() == t.Obj().Pkg() {
		return // skip dependencies in the same package
	}

	// add the dependency to the set
	if i.dep == nil {
		i.dep = make(map[*types.Named]struct{})
	}
	i.dep[t] = struct{}{}
}
