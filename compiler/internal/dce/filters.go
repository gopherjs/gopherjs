package dce

import (
	"go/types"
	"sort"
	"strconv"
	"strings"
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
func getFilters(o types.Object, tNest, tArgs []types.Type) (objectFilter, methodFilter string) {
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
				objectFilter = getObjectFilter(named.Obj(), tNest, tArgs)
			}

			// The method is not exported so we only need the method filter.
			if !o.Exported() {
				methodFilter = getMethodFilter(o, tArgs)
			}
			return
		}
	}

	// The object is not a method so we only need the object filter.
	objectFilter = getObjectFilter(o, tNest, tArgs)
	return
}

// getObjectFilter returns the object filter that functions as the primary
// name when determining if a declaration is alive or not.
// See [naming design] for more information.
//
// [naming design]: https://github.com/gopherjs/gopherjs/compiler/internal/dce/README.md#naming
func getObjectFilter(o types.Object, tNest, tArgs []types.Type) string {
	// TODO(grantnelson-wf): This needs to be resolving for nesting types too.
	return (&filterGen{argTypeRemap: tArgs}).Object(o, tArgs)
}

// getMethodFilter returns the method filter that functions as the secondary
// name when determining if a declaration is alive or not.
// See [naming design] for more information.
//
// [naming design]: https://github.com/gopherjs/gopherjs/compiler/internal/dce/README.md#naming
func getMethodFilter(o types.Object, tArgs []types.Type) string {
	if sig, ok := o.Type().(*types.Signature); ok {
		if len(tArgs) == 0 {
			if recv := sig.Recv(); recv != nil {
				tArgs = getTypeArgs(recv.Type())
			}
		}
		gen := &filterGen{argTypeRemap: tArgs}
		return objectName(o) + gen.Signature(sig)
	}
	return ``
}

// objectName returns the name part of a filter name,
// including the package path, if available.
//
// This is different from `o.Id` since it always includes the package path
// when available and doesn't add "_." when not available.
func objectName(o types.Object) string {
	if o.Pkg() != nil {
		return o.Pkg().Path() + `.` + o.Name()
	}
	return o.Name()
}

// getTypeArgs gets the type arguments for the given type
// wether they are type arguments or type parameters.
func getTypeArgs(typ types.Type) []types.Type {
	switch t := typ.(type) {
	case *types.Pointer:
		return getTypeArgs(t.Elem())
	case *types.Named:
		if typeArgs := t.TypeArgs(); typeArgs != nil {
			return typeListToSlice(typeArgs)
		}
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
		tParams[i] = typeParams.At(i).Constraint()
	}
	return tParams
}

type processingGroup struct {
	o     types.Object
	tArgs []types.Type
}

func (p processingGroup) is(o types.Object, tArgs []types.Type) bool {
	if len(p.tArgs) != len(tArgs) || p.o != o {
		return false
	}
	for i, tArg := range tArgs {
		if p.tArgs[i] != tArg {
			return false
		}
	}
	return true
}

type filterGen struct {
	// argTypeRemap is the type arguments in the same order as the
	// type parameters in the top level object such that the type parameters
	// index can be used to get the type argument.
	argTypeRemap []types.Type
	inProgress   []processingGroup
}

func (gen *filterGen) startProcessing(o types.Object, tArgs []types.Type) bool {
	for _, p := range gen.inProgress {
		if p.is(o, tArgs) {
			return false
		}
	}
	gen.inProgress = append(gen.inProgress, processingGroup{o, tArgs})
	return true
}

func (gen *filterGen) stopProcessing() {
	gen.inProgress = gen.inProgress[:len(gen.inProgress)-1]
}

// Object returns an object filter or filter part for an object.
func (gen *filterGen) Object(o types.Object, tArgs []types.Type) string {
	filter := objectName(o)

	// Add additional type information for generics and instances.
	if len(tArgs) == 0 {
		tArgs = getTypeArgs(o.Type())
	}
	if len(tArgs) > 0 {
		// Avoid infinite recursion in type arguments by
		// tracking the current object and type arguments being processed
		// and skipping if already in progress.
		if gen.startProcessing(o, tArgs) {
			filter += gen.TypeArgs(tArgs)
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
// arguments, e.g. `[any,int|string]`.
func (gen *filterGen) TypeArgs(tArgs []types.Type) string {
	parts := make([]string, len(tArgs))
	for i, tArg := range tArgs {
		parts[i] = gen.Type(tArg)
	}
	return `[` + strings.Join(parts, `, `) + `]`
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
	case types.Object:
		return gen.Object(t, nil)

	case *types.Array:
		return `[` + strconv.FormatInt(t.Len(), 10) + `]` + gen.Type(t.Elem())
	case *types.Chan:
		return `chan ` + gen.Type(t.Elem())
	case *types.Interface:
		return gen.Interface(t)
	case *types.Map:
		return `map[` + gen.Type(t.Key()) + `]` + gen.Type(t.Elem())
	case *types.Named:
		// Get type args from named instance not generic object
		return gen.Object(t.Obj(), getTypeArgs(t))
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
// If there is an argument remap, it will use the remapped type
// so long as it doesn't map to itself.
func (gen *filterGen) TypeParam(t *types.TypeParam) string {
	// TODO(grantnelson-wf): This needs to be resolving for nesting types too.
	index := t.Index()
	if index >= 0 && index < len(gen.argTypeRemap) {
		if inst := gen.argTypeRemap[index]; inst != t {
			return gen.Type(inst)
		}
	}
	if t.Constraint() == nil {
		return `any`
	}
	return gen.Type(t.Constraint())
}
