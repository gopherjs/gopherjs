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
	// concrete named type, such as functions.
	// Methods will use their receiver as the named type.
	name *types.Named

	// dep is a set of named types from other packages that this declaration
	// depends on. This may be empty if there are no dependencies.
	dep map[*types.Named]struct{}
}

// SetInstance sets the types and dependencies used by the grouper to represent
// the declaration this grouper info is attached to.
func (i *Info) SetInstance(tc *types.Context, inst typeparams.Instance) {
	i.setType(tc, inst)

	var pkg *types.Package
	if inst.Object != nil {
		pkg = inst.Object.Pkg()
	}

	i.addAllDeps(tc, inst, pkg)
}

func (i *Info) setType(tc *types.Context, inst typeparams.Instance) {
	if inst.Object == nil {
		return
	}
	switch t := inst.Object.Type().(type) {
	case *types.Named:
		i.name = inst.Resolve(tc).(*types.Named)
	case *types.Signature:
		if recv := typesutil.RecvType(t); recv != nil {
			inst2 := typeparams.Instance{
				Object: recv.Obj(),
				TNest:  inst.TNest,
				TArgs:  inst.TArgs,
			}
			i.name = inst2.Resolve(tc).(*types.Named)
		}
	}
}

func (i *Info) initPendingDeps(tc *types.Context, inst typeparams.Instance) []types.Type {
	var pending []types.Type
	pending = append(pending, inst.TNest...)
	pending = append(pending, inst.TArgs...)

	if inst.Object == nil {
		// shouldn't happen, but if it does, just check the type args.
		return pending
	}

	switch t := inst.Object.Type().(type) {
	case *types.Named:
		r := typeparams.NewResolver(tc, inst)
		tArgs := r.Substitute(t).(*types.Named).TypeArgs()
		for j := tArgs.Len() - 1; j >= 0; j-- {
			pending = append(pending, tArgs.At(j))
		}
		pending = append(pending, r.Substitute(t.Underlying()))
		for j := t.NumMethods() - 1; j >= 0; j-- {
			pending = append(pending, r.Substitute(t.Method(j).Type()))
		}
		return pending

	case *types.Signature:
		if recv := typesutil.RecvType(t); recv != nil {
			// The instance is a method, resolve the receiver type
			// and the signature of the method to find its dependencies.
			recvInst := typeparams.Instance{
				Object: recv.Obj(),
				TNest:  inst.TNest,
				TArgs:  inst.TArgs,
			}
			r := typeparams.NewResolver(tc, recvInst)
			pending = append(pending, r.Substitute(t))
			return pending
		}

		// The instance is a function, resolve the signature.
		pending = append(pending, inst.Resolve(tc))
		return pending

	default:
		// If not a named type or method, we can add the type
		// as a dependency directly without needing to resolve it further.
		// This will take a type like `[]Cat` and add `Cat` as a dependency.
		pending = append(pending, inst.Object.Type())
		return pending
	}
}

func (i *Info) addAllDeps(tc *types.Context, inst typeparams.Instance, pkg *types.Package) {
	pending := i.initPendingDeps(tc, inst)
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
			if t.Obj() == nil || t.Obj().Pkg() == nil {
				continue // skip objects in universal scope, e.g. `error`
			}
			if typesutil.IsJsPackage(t.Obj().Pkg()) && t.Obj().Name() == "Object" {
				continue // skip *js.Object
			}

			// Add the basics for this object including methods.
			inst2 := typeparams.Instance{Object: t.Obj()}
			tArgs := t.TypeArgs()
			inst2.TArgs = make(typesutil.TypeList, tArgs.Len())
			for j := tArgs.Len() - 1; j >= 0; j-- {
				inst2.TArgs[j] = tArgs.At(j)
			}
			r := typeparams.NewResolver(tc, inst2)
			for j := t.NumMethods() - 1; j >= 0; j-- {
				pending = append(pending, r.Substitute(t.Method(j).Type()))
			}

			if pkg != nil && pkg == t.Obj().Pkg() {
				// Skip over named types from the same package,
				// continue into them to depend on the same dependencies as they do.
				// This prevents circular dependencies from being added.
				for j := tArgs.Len() - 1; j >= 0; j-- {
					pending = append(pending, tArgs.At(j))
				}
				pending = append(pending, r.Substitute(t.Underlying()))
				continue
			}

			// add the dependency to the set
			if i.dep == nil {
				i.dep = make(map[*types.Named]struct{})
			}
			i.dep[t] = struct{}{}

		case *types.Interface:
			for j := t.NumExplicitMethods() - 1; j >= 0; j-- {
				pending = append(pending, t.ExplicitMethod(j).Type())
			}
			for j := t.NumEmbeddeds() - 1; j >= 0; j-- {
				pending = append(pending, t.EmbeddedType(j))
			}

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
