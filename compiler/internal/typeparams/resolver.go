package typeparams

import (
	"fmt"
	"go/types"
	"sort"
	"strings"

	"github.com/gopherjs/gopherjs/compiler/typesutil"
	"github.com/gopherjs/gopherjs/internal/govendor/subst"
)

// Resolver translates types defined in terms of type parameters into concrete
// types, given a root instance. The root instance provides context for mapping
// from type parameters to type arguments so that the resolver can substitute
// any type parameters used in types to the corresponding type arguments.
//
// In some cases, a generic type may not be able to be fully instantiated.
// Generic named types that have no type arguments applied will have the
// type parameters substituted, however the type arguments will not be
// applied to instantiate the named type.
//
// For example, given `func Foo[T any]() { type Bar[U *T] struct { x T; y U } }`,
// and if `Foo[int]` is used as the root for the resolver, then `Bar[U *T]` will
// be substituted to create the generic `Bar[U *int] struct { x int; y U }`.
// Alternatively, the instantiated but still generic because of the `T`,
// `Bar[bool] struct { x T; y bool}` will be substituted for `Foo[int]` to
// create the concrete `Bar[bool] struct { x int; y bool }`.
//
// Typically the instantiated type from `info.Instances` should be substituted
// to resolve the implicit nesting types and create a concrete type.
// See internal/govendor/subst/subst.go for more details.
type Resolver struct {
	tParams      *types.TypeParamList
	tArgs        []types.Type
	nest         *types.Func
	nestTParams  *types.TypeParamList
	nestTArgs    []types.Type
	replacements map[*types.TypeParam]types.Type
	root         Instance

	// subster is the substitution helper that will perform the actual
	// substitutions. This maybe nil when there are no substitutions but
	// will still be usable when nil.
	subster *subst.Subster
	selMemo map[typesutil.Selection]typesutil.Selection
}

// NewResolver creates a new Resolver that will substitute type parameters
// with the type arguments as defined in the provided Instance.
func NewResolver(tc *types.Context, root Instance) *Resolver {
	var (
		nest         *types.Func
		nestTParams  *types.TypeParamList
		tParams      *types.TypeParamList
		replacements = map[*types.TypeParam]types.Type{}
	)

	switch typ := root.Object.Type().(type) {
	case *types.Signature:
		nest, _ = root.Object.(*types.Func)
		tParams = SignatureTypeParams(typ)
	case *types.Named:
		tParams = typ.TypeParams()
		nest = FindNestingFunc(root.Object)
		if nest != nil {
			nestTParams = SignatureTypeParams(nest.Type().(*types.Signature))
		}
	default:
		panic(fmt.Errorf("unexpected type %T for object %s", typ, root.Object))
	}

	// Check the root's implicit nesting type parameters and arguments match,
	// then add them to the replacements.
	if nestTParams.Len() != len(root.TNest) {
		panic(fmt.Errorf(`number of nesting type parameters and arguments must match: %d => %d`, nestTParams.Len(), len(root.TNest)))
	}
	for i := 0; i < nestTParams.Len(); i++ {
		replacements[nestTParams.At(i)] = root.TNest[i]
	}

	// If no type arguments are provided, check if the type already has
	// type arguments. This is the case for instantiated objects in the instance.
	if tParams.Len() > 0 && len(root.TArgs) == 0 {
		if typ, ok := root.Object.Type().(interface{ TypeArgs() *types.TypeList }); ok {
			root.TArgs = make(typesutil.TypeList, typ.TypeArgs().Len())
			for i := 0; i < typ.TypeArgs().Len(); i++ {
				root.TArgs[i] = typ.TypeArgs().At(i)
			}
		}
	}

	// Check the root's type parameters and arguments match,
	// then add them to the replacements.
	if tParams.Len() != len(root.TArgs) {
		panic(fmt.Errorf(`number of type parameters and arguments must match: %d => %d for %s`, tParams.Len(), len(root.TArgs), root.String()))
	}
	for i := 0; i < tParams.Len(); i++ {
		replacements[tParams.At(i)] = root.TArgs[i]
	}

	return &Resolver{
		tParams:      tParams,
		tArgs:        root.TArgs,
		nest:         nest,
		nestTParams:  nestTParams,
		nestTArgs:    root.TNest,
		replacements: replacements,
		root:         root,
		subster:      subst.New(tc, replacements),
		selMemo:      map[typesutil.Selection]typesutil.Selection{},
	}
}

// TypeParams is the list of type parameters that this resolver will substitute.
func (r *Resolver) TypeParams() *types.TypeParamList {
	if r == nil {
		return nil
	}
	return r.tParams
}

// TypeArgs is the list of type arguments that this resolver will resolve to.
func (r *Resolver) TypeArgs() []types.Type {
	if r == nil {
		return nil
	}
	return r.tArgs
}

// Nest is the nesting function that this resolver will resolve types with.
// This will be null if the resolver is not for a nested context,
func (r *Resolver) Nest() *types.Func {
	if r == nil {
		return nil
	}
	return r.nest
}

// NestTypeParams is the list of type parameters from the nesting function
// that this resolver will substitute.
func (r *Resolver) NestTypeParams() *types.TypeParamList {
	if r == nil {
		return nil
	}
	return r.nestTParams
}

// NestTypeArgs is the list of type arguments from the nesting function
// that this resolver will resolve to.
func (r *Resolver) NestTypeArgs() []types.Type {
	if r == nil {
		return nil
	}
	return r.nestTArgs
}

// Substitute replaces references to type params in the provided type definition
// with the corresponding concrete types.
func (r *Resolver) Substitute(typ types.Type) types.Type {
	if r == nil || typ == nil {
		return typ // No substitutions to be made.
	}
	return r.subster.Type(typ)
}

// SubstituteAll same as Substitute, but accepts a TypeList are returns
// substitution results as a slice in the same order.
func (r *Resolver) SubstituteAll(list *types.TypeList) []types.Type {
	result := make([]types.Type, list.Len())
	for i := range result {
		result[i] = r.Substitute(list.At(i))
	}
	return result
}

// SubstituteSelection replaces a method of field selection on a generic type
// defined in terms of type parameters with a method selection on a concrete
// instantiation of the type.
func (r *Resolver) SubstituteSelection(sel typesutil.Selection) typesutil.Selection {
	if r == nil || sel == nil {
		return sel // No substitutions to be made.
	}
	if concrete, ok := r.selMemo[sel]; ok {
		return concrete
	}

	switch sel.Kind() {
	case types.MethodExpr, types.MethodVal, types.FieldVal:
		recv := r.Substitute(sel.Recv())
		if types.Identical(recv, sel.Recv()) {
			return sel // Non-generic receiver, no substitution necessary.
		}

		// Look up the method on the instantiated receiver.
		pkg := sel.Obj().Pkg()
		obj, index, _ := types.LookupFieldOrMethod(recv, true, pkg, sel.Obj().Name())
		if obj == nil {
			panic(fmt.Errorf("failed to lookup field %q in type %v", sel.Obj().Name(), recv))
		}
		typ := obj.Type()

		if sel.Kind() == types.MethodExpr {
			typ = typesutil.RecvAsFirstArg(typ.(*types.Signature))
		}
		concrete := typesutil.NewSelection(sel.Kind(), recv, index, obj, typ)
		r.selMemo[sel] = concrete
		return concrete
	default:
		panic(fmt.Errorf("unexpected selection kind %v: %v", sel.Kind(), sel))
	}
}

// String gets a strings representation of the resolver for debugging.
func (r *Resolver) String() string {
	if r == nil {
		return `{}`
	}

	parts := make([]string, 0, len(r.replacements))
	for tp, ta := range r.replacements {
		parts = append(parts, fmt.Sprintf("%s->%s", tp, ta))
	}
	sort.Strings(parts)
	return `{` + strings.Join(parts, `, `) + `}`
}
