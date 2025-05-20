package typeparams

import (
	"fmt"
	"go/types"
	"strings"

	"github.com/gopherjs/gopherjs/compiler/typesutil"
	"github.com/gopherjs/gopherjs/internal/govendor/subst"
)

// Resolver translates types defined in terms of type parameters into concrete
// types, given a mapping from type params to type arguments.
type Resolver struct {
	tParams *types.TypeParamList
	tArgs   []types.Type
	parent  *Resolver

	// subster is the substitution helper that will perform the actual
	// substitutions. This maybe nil when there are no substitutions but
	// will still be usable when nil.
	subster *subst.Subster
	selMemo map[typesutil.Selection]typesutil.Selection
}

// NewResolver creates a new Resolver with tParams entries mapping to tArgs
// entries with the same index.
func NewResolver(tc *types.Context, tParams *types.TypeParamList, tArgs []types.Type, parent *Resolver) *Resolver {
	r := &Resolver{
		tParams: tParams,
		tArgs:   tArgs,
		parent:  parent,
		subster: subst.New(tc, tParams, tArgs),
		selMemo: map[typesutil.Selection]typesutil.Selection{},
	}
	return r
}

// TypeParams is the list of type parameters that this resolver will substitute.
// This will not including any implicit type parameters from a nesting function or method.
func (r *Resolver) TypeParams() *types.TypeParamList {
	if r == nil {
		return nil
	}
	return r.tParams
}

// TypeArgs is the list of type arguments that this resolver will resolve to.
// This will not including any implicit type parameters from a nesting function or method.
func (r *Resolver) TypeArgs() []types.Type {
	if r == nil {
		return nil
	}
	return r.tArgs
}

// Substitute replaces references to type params in the provided type definition
// with the corresponding concrete types.
func (r *Resolver) Substitute(typ types.Type) types.Type {
	if r == nil || typ == nil {
		return typ // No substitutions to be made.
	}
	typ = r.subster.Type(typ)
	typ = r.parent.Substitute(typ)
	return typ
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

	parts := make([]string, 0, len(r.tArgs))
	for i, ta := range r.tArgs {
		parts = append(parts, fmt.Sprintf("%s->%s", r.tParams.At(i), ta))
	}

	nestStr := ``
	if r.parent != nil {
		nestStr = r.parent.String() + `:`
	}
	return nestStr + `{` + strings.Join(parts, `, `) + `}`
}
