package dce

import (
	"go/types"
	"sort"
	"strconv"
	"strings"

	"github.com/gopherjs/gopherjs/compiler/internal/typeparams"
	"github.com/gopherjs/gopherjs/compiler/typesutil"
)

// getFilters determines the DCE filters for the given object.
// This will return an object filter and optionally return a method filter.
//
// Typically, the object filter will always be set and the method filter
// will be empty unless the object is an unexported method.
// However, when the object is a method invocation on an unnamed interface type
// the object filter will be empty and only the method filter will be set.
// The later shouldn't happen when naming a declaration but only when creating
// dependencies.
func getFilters(o types.Object, nestTArgs, tArgs []types.Type) (objectFilter, methodFilter string) {
	if f, ok := o.(*types.Func); ok {
		sig := f.Type().(*types.Signature)
		if recv := sig.Recv(); recv != nil {
			// The object is a method so the object filter is the receiver type
			// if the receiver type is named, otherwise it's an unnamed interface.
			typ := recv.Type()
			if ptrType, ok := typ.(*types.Pointer); ok {
				typ = ptrType.Elem()
			}
			if len(tArgs) == 0 {
				tArgs = getTypeArgs(typ)
			}
			if named, ok := typ.(*types.Named); ok {
				objectFilter = getObjectFilter(named.Obj(), nil, tArgs)
			}

			// The method is not exported so we only need the method filter.
			if !o.Exported() {
				methodFilter = getMethodFilter(o, tArgs)
			}
			return
		}
	}

	// The object is not a method so we only need the object filter.
	objectFilter = getObjectFilter(o, nestTArgs, tArgs)
	return
}

// getObjectFilter returns the object filter that functions as the primary
// name when determining if a declaration is alive or not.
// See [naming design] for more information.
//
// [naming design]: https://github.com/gopherjs/gopherjs/compiler/internal/dce/README.md#naming
func getObjectFilter(o types.Object, nestTArgs, tArgs []types.Type) string {
	return (&filterGen{}).Object(o, nestTArgs, tArgs)
}

// getMethodFilter returns the method filter that functions as the secondary
// name when determining if a declaration is alive or not.
// See [naming design] for more information.
//
// [naming design]: https://github.com/gopherjs/gopherjs/compiler/internal/dce/README.md#naming
func getMethodFilter(o types.Object, tArgs []types.Type) string {
	if sig, ok := o.Type().(*types.Signature); ok {
		gen := &filterGen{}
		tParams := getTypeParams(o.Type())
		if len(tArgs) == 0 {
			tArgs = getTypeArgs(sig)
		}
		if len(tArgs) > 0 {
			gen.addReplacements(tParams, tArgs)
		}
		return objectName(o) + gen.Signature(sig)
	}
	return ``
}

// objectName returns the name part of a filter name,
// including the package path and nest names, if available.
//
// This is different from `o.Id` since it always includes the package path
// when available and doesn't add "_." when not available.
func objectName(o types.Object) string {
	prefix := ``
	if o.Pkg() != nil {
		prefix += o.Pkg().Path() + `.`
	}
	if nest := typeparams.FindNestingFunc(o); nest != nil && nest != o {
		if recv := typesutil.RecvType(nest.Type().(*types.Signature)); recv != nil {
			prefix += recv.Obj().Name() + `:`
		}
		prefix += nest.Name() + `:`
	}
	return prefix + o.Name()
}

// getNestTypeParams gets the type parameters for the nesting function
// or nil if the object is not nested in a function or
// the given object is a function itself.
func getNestTypeParams(o types.Object) []types.Type {
	fn := typeparams.FindNestingFunc(o)
	if fn == nil || fn == o {
		return nil
	}

	tp := typeparams.SignatureTypeParams(fn.Type().(*types.Signature))
	nestTParams := make([]types.Type, tp.Len())
	for i := 0; i < tp.Len(); i++ {
		nestTParams[i] = tp.At(i)
	}
	return nestTParams
}

// getTypeArgs gets the type arguments for the given type
// or nil if the type does not have type arguments.
func getTypeArgs(typ types.Type) []types.Type {
	switch t := typ.(type) {
	case *types.Pointer:
		return getTypeArgs(t.Elem())
	case *types.Named:
		return typeListToSlice(t.TypeArgs())
	}
	return nil
}

// getTypeParams gets the type parameters for the given type
// or nil if the type does not have type parameters.
func getTypeParams(typ types.Type) []types.Type {
	switch t := typ.(type) {
	case *types.Pointer:
		return getTypeParams(t.Elem())
	case *types.Named:
		if typeParams := t.TypeParams(); typeParams != nil {
			return typeParamListToSlice(typeParams)
		}
	case *types.Signature:
		if typeParams := t.RecvTypeParams(); typeParams != nil {
			return typeParamListToSlice(typeParams)
		}
		if typeParams := t.TypeParams(); typeParams != nil {
			return typeParamListToSlice(typeParams)
		}
	}
	return nil
}

// typeListToSlice returns the list of type arguments for the type arguments.
func typeListToSlice(typeArgs *types.TypeList) []types.Type {
	tArgs := make([]types.Type, typeArgs.Len())
	for i := range tArgs {
		tArgs[i] = typeArgs.At(i)
	}
	return tArgs
}

// typeParamListToSlice returns the list of type arguments for the type parameters.
func typeParamListToSlice(typeParams *types.TypeParamList) []types.Type {
	tParams := make([]types.Type, typeParams.Len())
	for i := range tParams {
		tParams[i] = typeParams.At(i)
	}
	return tParams
}

type processingGroup struct {
	o         types.Object
	nestTArgs []types.Type
	tArgs     []types.Type
}

func (p processingGroup) is(o types.Object, nestTArgs, tArgs []types.Type) bool {
	if len(p.nestTArgs) != len(nestTArgs) || len(p.tArgs) != len(tArgs) || p.o != o {
		return false
	}
	for i, ta := range nestTArgs {
		if p.nestTArgs[i] != ta {
			return false
		}
	}
	for i, ta := range tArgs {
		if p.tArgs[i] != ta {
			return false
		}
	}
	return true
}

type filterGen struct {
	// replacement is used to use another type in place of a given type
	// this is typically used for type parameters to type arguments.
	replacement map[types.Type]types.Type
	inProgress  []processingGroup
}

// addReplacements adds a mapping from one type to another.
// The slices should be the same length but will ignore any extra types.
// Any replacement where the key and value are the same will be ignored.
func (gen *filterGen) addReplacements(from []types.Type, to []types.Type) {
	if len(from) == 0 || len(to) == 0 {
		return
	}

	if gen.replacement == nil {
		gen.replacement = map[types.Type]types.Type{}
	}

	count := len(from)
	if count > len(to) {
		count = len(to)
	}
	for i := 0; i < count; i++ {
		if from[i] != to[i] {
			gen.replacement[from[i]] = to[i]
		}
	}
}

// pushGenerics prepares the filter generator for processing an object
// by setting any generic information and nesting information needed for it.
// It returns the type arguments for the object and a function to restore
// the previous state of the filter generator.
func (gen *filterGen) pushGenerics(o types.Object, nestTArgs, tArgs []types.Type) ([]types.Type, []types.Type, func()) {
	// Create a new replacement map and copy the old one into it.
	oldReplacement := gen.replacement
	gen.replacement = map[types.Type]types.Type{}
	for tp, ta := range oldReplacement {
		gen.replacement[tp] = ta
	}

	// Prepare the nested context for the object.
	nestTParams := getNestTypeParams(o)
	if len(nestTArgs) > 0 {
		gen.addReplacements(nestTParams, nestTArgs)
	} else {
		nestTArgs = nestTParams
	}

	// Prepare the type arguments for the object.
	tParams := getTypeParams(o.Type())
	if len(tArgs) == 0 {
		tArgs = getTypeArgs(o.Type())
	}
	if len(tArgs) > 0 {
		gen.addReplacements(tParams, tArgs)
	} else {
		tArgs = tParams
	}

	// Return a function to restore the old replacement map.
	return nestTArgs, tArgs, func() {
		gen.replacement = oldReplacement
	}
}

func (gen *filterGen) startProcessing(o types.Object, nestTArgs, tArgs []types.Type) bool {
	for _, p := range gen.inProgress {
		if p.is(o, nestTArgs, tArgs) {
			return false
		}
	}
	gen.inProgress = append(gen.inProgress, processingGroup{o: o, nestTArgs: nestTArgs, tArgs: tArgs})
	return true
}

func (gen *filterGen) stopProcessing() {
	gen.inProgress = gen.inProgress[:len(gen.inProgress)-1]
}

// Object returns an object filter or filter part for an object.
func (gen *filterGen) Object(o types.Object, nestTArgs, tArgs []types.Type) string {
	filter := objectName(o)

	// Add additional type information for generics and instances.
	nestTArgs, tArgs, popGenerics := gen.pushGenerics(o, nestTArgs, tArgs)
	defer popGenerics()

	if len(tArgs) > 0 || len(nestTArgs) > 0 {
		// Avoid infinite recursion in type arguments by
		// tracking the current object and type arguments being processed
		// and skipping if already in progress.
		if gen.startProcessing(o, nestTArgs, tArgs) {
			filter += gen.TypeArgs(nestTArgs, tArgs)
			gen.stopProcessing()
		} else {
			filter += `[...]`
		}
	}

	return filter
}

// Signature returns the filter part containing the signature
// parameters and results for a function or method, e.g. `(int)(bool,error)`.
func (gen *filterGen) Signature(sig *types.Signature) string {
	filter := `(` + gen.Tuple(sig.Params(), sig.Variadic()) + `)`
	switch sig.Results().Len() {
	case 0:
		break
	case 1:
		filter += ` ` + gen.Type(sig.Results().At(0).Type())
	default:
		filter += `(` + gen.Tuple(sig.Results(), false) + `)`
	}
	return filter
}

// TypeArgs returns the filter part containing the type
// arguments, e.g. `[bool;any,int|string]`.
func (gen *filterGen) TypeArgs(nestTArgs, tArgs []types.Type) string {
	toStr := func(t []types.Type) string {
		parts := make([]string, len(t))
		for i, ta := range t {
			parts[i] = gen.Type(ta)
		}
		return strings.Join(parts, `, `)
	}

	head := `[`
	if len(nestTArgs) > 0 {
		head += toStr(nestTArgs) + `;`
		if len(tArgs) > 0 {
			head += ` `
		}
	}
	return head + toStr(tArgs) + `]`
}

// Tuple returns the filter part containing parameter or result
// types for a function, e.g. `(int,string)`, `(int,...string)`.
func (gen *filterGen) Tuple(t *types.Tuple, variadic bool) string {
	count := t.Len()
	parts := make([]string, count)
	for i := range parts {
		argType := t.At(i).Type()
		if i == count-1 && variadic {
			if slice, ok := argType.(*types.Slice); ok {
				argType = slice.Elem()
			}
			parts[i] = `...` + gen.Type(argType)
		} else {
			parts[i] = gen.Type(argType)
		}
	}
	return strings.Join(parts, `, `)
}

// Type returns the filter part for a single type.
func (gen *filterGen) Type(typ types.Type) string {
	switch t := typ.(type) {
	case *types.Array:
		return `[` + strconv.FormatInt(t.Len(), 10) + `]` + gen.Type(t.Elem())
	case *types.Chan:
		return `chan ` + gen.Type(t.Elem())
	case *types.Interface:
		return gen.Interface(t)
	case *types.Map:
		return `map[` + gen.Type(t.Key()) + `]` + gen.Type(t.Elem())
	case *types.Named:
		// Get type args from named instance not generic object.
		return gen.Object(t.Obj(), nil, typeListToSlice(t.TypeArgs()))
	case *types.Pointer:
		return `*` + gen.Type(t.Elem())
	case *types.Signature:
		return `func` + gen.Signature(t)
	case *types.Slice:
		return `[]` + gen.Type(t.Elem())
	case *types.Struct:
		return gen.Struct(t)
	case *types.TypeParam:
		return gen.TypeParam(t)
	default:
		// Anything else, like basics, just stringify normally.
		return t.String()
	}
}

// Union returns the filter part for a union of types from an type parameter
// constraint, e.g. `~string|int|~float64`.
func (gen *filterGen) Union(u *types.Union) string {
	parts := make([]string, u.Len())
	for i := range parts {
		term := u.Term(i)
		part := gen.Type(term.Type())
		if term.Tilde() {
			part = "~" + part
		}
		parts[i] = part
	}
	// Sort the union so that "string|int" matches "int|string".
	sort.Strings(parts)
	return strings.Join(parts, `|`)
}

// Interface returns the filter part for an interface type or
// an interface for a type parameter constraint.
func (gen *filterGen) Interface(inter *types.Interface) string {
	// Collect all method constraints with method names and signatures.
	parts := make([]string, inter.NumMethods())
	for i := range parts {
		fn := inter.Method(i)
		parts[i] = fn.Id() + gen.Signature(fn.Type().(*types.Signature))
	}
	// Add any union constraints.
	for i := 0; i < inter.NumEmbeddeds(); i++ {
		if union, ok := inter.EmbeddedType(i).(*types.Union); ok {
			parts = append(parts, gen.Union(union))
		}
	}
	// Sort the parts of the interface since the order doesn't matter.
	// e.g. `interface { a(); b() }` is the same as `interface { b(); a() }`.
	sort.Strings(parts)

	if len(parts) == 0 {
		return `any`
	}
	if inter.NumMethods() == 0 && len(parts) == 1 {
		return parts[0] // single constraint union, i.e. `bool|~int|string`
	}
	return `interface{ ` + strings.Join(parts, `; `) + ` }`
}

// Struct returns the filter part for a struct type.
func (gen *filterGen) Struct(s *types.Struct) string {
	if s.NumFields() == 0 {
		return `struct{}`
	}
	parts := make([]string, s.NumFields())
	for i := range parts {
		f := s.Field(i)
		// The field name and order is required to be part of the filter since
		// struct matching rely on field names too. Tags are not needed.
		// See https://go.dev/ref/spec#Conversions
		parts[i] = f.Id() + ` ` + gen.Type(f.Type())
	}
	return `struct{ ` + strings.Join(parts, `; `) + ` }`
}

// TypeParam returns the filter part for a type parameter.
// If there is an argument remap, it will use the remapped type.
func (gen *filterGen) TypeParam(t *types.TypeParam) string {
	if inst, exists := gen.replacement[t]; exists {
		return gen.Type(inst)
	}
	if t.Constraint() == nil {
		return `any`
	}
	return gen.Type(t.Constraint())
}
