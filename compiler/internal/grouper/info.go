package grouper

import (
	"fmt"
	"go/types"
	"sort"
	"strings"

	"github.com/gopherjs/gopherjs/compiler/internal/typeparams"
	"github.com/gopherjs/gopherjs/compiler/typesutil"
)

type Info struct {
	// Group is the group number for initializing this declaration's DeclCode.
	// The declarations in the same group should still be initialized in the
	// same order as they were declared based on imports first.
	Group int

	// name is the concrete named type that this declaration is associated with.
	// This may be nil for declarations that do not have an associated concrete
	// named type, such as functions, or have a skipped type, like `error`.
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

func (i *Info) String() string {
	name := `<unnamed>`
	if i.name != nil {
		name = i.name.String()
	}
	deps := make([]string, 0, len(i.dep))
	for dep := range i.dep {
		deps = append(deps, dep.String())
	}
	sort.Strings(deps)
	return fmt.Sprintf(`Info(%d, %s, [%s])`, i.Group, name, strings.Join(deps, `, `))
}

func skipType(t *types.Named) bool {
	if t.Obj() == nil || t.Obj().Pkg() == nil {
		return true // skip objects in universal scope, e.g. `error`
	}
	if typesutil.IsJsPackage(t.Obj().Pkg()) {
		return true // skip js.Object and js.Error
	}
	return false
}

func (i *Info) setType(tc *types.Context, inst typeparams.Instance) {
	if inst.Object == nil {
		return
	}
	if _, ok := inst.Object.Type().(*types.Named); ok {
		if name := inst.Resolve(tc).(*types.Named); name != nil && !skipType(name) {
			i.name = name
		}
	}
}

type dedupStack struct {
	stack []types.Type
	seen  map[types.Type]struct{}
}

func (s *dedupStack) push(t ...types.Type) {
	for _, t := range t {
		if _, dup := s.seen[t]; t != nil && !dup {
			s.seen[t] = struct{}{}
			s.stack = append(s.stack, t)
		}
	}
}

func (s *dedupStack) pop() types.Type {
	if maxIndex := len(s.stack) - 1; maxIndex >= 0 {
		t := s.stack[maxIndex]
		s.stack = s.stack[:maxIndex]
		return t
	}
	return nil
}

func (s *dedupStack) hasMore() bool {
	return len(s.stack) > 0
}

func (i *Info) addAllDeps(tc *types.Context, inst typeparams.Instance, pkg *types.Package) {
	if inst.Object == nil {
		return
	}

	pending := &dedupStack{
		seen: make(map[types.Type]struct{}),
	}
	pending.push(inst.TNest...)
	pending.push(inst.TArgs...)

	switch t := inst.Object.Type().(type) {
	case *types.Named:
		r := typeparams.NewResolver(tc, inst)
		tArgs := r.Substitute(t).(*types.Named).TypeArgs()
		for j := tArgs.Len() - 1; j >= 0; j-- {
			pending.push(tArgs.At(j))
		}
		pending.push(r.Substitute(t.Underlying()))
	default:
		// If not a named type or method, we can add the type
		// as a dependency directly without needing to resolve it further.
		// This will take a type like `[]Cat` and add `Cat` as a dependency.
		pending.push(inst.Object.Type())
	}

	for pending.hasMore() {
		t := pending.pop()

		switch t := t.(type) {
		case *types.Basic:
			// ignore basic types like int, string, unsafe.Pointer, etc.

		case *types.Named:
			if skipType(t) {
				continue
			}

			if pkg != nil && pkg == t.Obj().Pkg() {
				// Skip over named types from the same package,
				// continue into them to depend on the same dependencies as they do.
				// This prevents circular dependencies in one package from being added.

				inst2 := typeparams.Instance{Object: t.Obj()}
				tArgs := t.TypeArgs()
				inst2.TArgs = make(typesutil.TypeList, tArgs.Len())
				for j := range inst2.TArgs {
					inst2.TArgs[j] = tArgs.At(j)
					pending.push(tArgs.At(j))
				}
				r := typeparams.NewResolver(tc, inst2)
				pending.push(r.Substitute(t.Underlying()))
				continue
			}

			// add the dependency to the set
			if i.dep == nil {
				i.dep = make(map[*types.Named]struct{})
			}
			i.dep[t] = struct{}{}

		// TODO(grantnelson-wf): REMOVE OR KEEP (check anonTypes)
		//case *types.Interface:
		//	for j := t.NumExplicitMethods() - 1; j >= 0; j-- {
		//		pending.push(t.ExplicitMethod(j).Type())
		//	}
		//	for j := t.NumEmbeddeds() - 1; j >= 0; j-- {
		//		pending.push(t.EmbeddedType(j))
		//	}

		// TODO(grantnelson-wf): REMOVE OR KEEP (check anonTypes)
		//case *types.Struct:
		//	for j := t.NumFields() - 1; j >= 0; j-- {
		//		pending.push(t.Field(j).Type())
		//	}

		// TODO(grantnelson-wf): REMOVE OR KEEP (check anonTypes)
		case *types.Signature:
			for j := t.Params().Len() - 1; j >= 0; j-- {
				pending.push(t.Params().At(j).Type())
			}
			for j := t.Results().Len() - 1; j >= 0; j-- {
				pending.push(t.Results().At(j).Type())
			}

		case *types.Map:
			pending.push(t.Key())
			pending.push(t.Elem())

		case interface{ Elem() types.Type }:
			// Handles *types.Pointer, *types.Slice, *types.Array, and *types.Chan
			pending.push(t.Elem())
		}
	}
}
