package typeparams

import (
	"fmt"
	"go/ast"
	"go/types"
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
// For example, given `func Foo[T any]() { type Bar[U *T] struct { x T; y U } }`,
// and if `Foo[int]` is used as the root for the resolver, then `Bar[U *T]` will
// be substituted to create the generic `Bar[U *int] struct { x int; y U }`,
// and the generic (because of the `T`) `Bar[bool] struct { x T; y bool}` will
// be substituted to create the concrete `Bar[bool] struct { x int; y bool }`.
// Typically the instantiated type from `info.Instances` should be substituted
// to resolve the implicit nesting types and create a concrete type.
// See internal/govendor/subst/subst.go for more details.
type Resolver struct {
	tc      *types.Context
	tParams *types.TypeParamList
	tArgs   []types.Type
	root    Instance

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
		fn           *types.Func
		nestTParams  *types.TypeParamList
		tParams      *types.TypeParamList
		replacements = map[*types.TypeParam]types.Type{}
	)

	switch typ := root.Object.Type().(type) {
	case *types.Signature:
		fn = root.Object.(*types.Func)
		tParams = SignatureTypeParams(typ)
	case *types.Named:
		fn = FindNestingFunc(root.Object)
		tParams = typ.TypeParams()
		if fn != nil {
			nestTParams = SignatureTypeParams(fn.Type().(*types.Signature))
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

	// Check the root's type parameters and arguments match,
	// then add them to the replacements.
	if tParams.Len() != len(root.TArgs) {
		panic(fmt.Errorf(`number of type parameters and arguments must match: %d => %d`, tParams.Len(), len(root.TArgs)))
	}
	for i := 0; i < tParams.Len(); i++ {
		replacements[tParams.At(i)] = root.TArgs[i]
	}

	return &Resolver{
		tc:      tc,
		tParams: tParams,
		tArgs:   root.TArgs,
		root:    root,
		subster: subst.New(tc, fn, replacements),
		selMemo: map[typesutil.Selection]typesutil.Selection{},
	}
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

	parts := make([]string, 0, len(r.tArgs))
	for i, ta := range r.tArgs {
		parts = append(parts, fmt.Sprintf("%s->%s", r.tParams.At(i), ta))
	}
	return `{` + strings.Join(parts, `, `) + `}`
}

// visitor implements ast.Visitor to collect instances of generic types and
// functions into an InstanceSet.
//
// When traversing an AST subtree corresponding to a generic type, method or
// function, Resolver must be provided mapping the type parameters into concrete
// types.
type visitor struct {
	instances *PackageInstanceSets
	resolver  *Resolver
	info      *types.Info
	tNest     []types.Type // The type arguments for a nested context.
}

var _ ast.Visitor = &visitor{}

func (c *visitor) Visit(n ast.Node) ast.Visitor {
	if ident, ok := n.(*ast.Ident); ok {
		c.visitIdent(ident)
	}
	return c
}

func (c *visitor) visitIdent(ident *ast.Ident) {
	if inst, ok := c.info.Instances[ident]; ok {
		// Found the use of a generic type or function.
		c.visitInstance(ident, inst)
	}

	if len(c.resolver.TypeArgs()) > 0 {
		if obj, ok := c.info.Defs[ident]; ok && obj != nil {
			// Found instance of a type defined inside a generic context.
			c.visitNestedType(obj)
		}
	}
}

func (c *visitor) visitInstance(ident *ast.Ident, inst types.Instance) {
	obj := c.info.Uses[ident]
	tArgs := inst.TypeArgs

	// For types embedded in structs, the object the identifier resolves to is a
	// *types.Var representing the implicitly declared struct field. However, the
	// instance relates to the *types.TypeName behind the field type, which we
	// obtain here.
	typ := obj.Type()
	if ptr, ok := typ.(*types.Pointer); ok {
		typ = ptr.Elem()
	}
	if t, ok := typ.(*types.Named); ok {
		obj = t.Obj()
	}

	// If the object is defined in the same scope as the instance,
	// then we apply the current nested type arguments.
	var tNest []types.Type
	if obj.Parent().Contains(ident.Pos()) {
		tNest = c.tNest
	}

	c.addInstance(obj, tArgs, tNest)
}

func (c *visitor) visitNestedType(obj types.Object) {
	if _, ok := obj.(*types.TypeName); !ok {
		// Found a variable or function, not a type, so skip it.
		return
	}

	typ := obj.Type()
	if ptr, ok := typ.(*types.Pointer); ok {
		typ = ptr.Elem()
	}

	t, ok := typ.(*types.Named)
	if !ok || t.TypeParams().Len() > 0 {
		// Found a generic type or an unnamed type (e.g. type parameter).
		// Don't add generic types yet because they
		// will be added when we find an instance of them.
		return
	}

	c.addInstance(obj, nil, c.resolver.TypeArgs())
}

func (c *visitor) addInstance(obj types.Object, tArgList *types.TypeList, tNest []types.Type) {
	tArgs := c.resolver.SubstituteAll(tArgList)
	if isGeneric(tArgs...) {
		// Skip any instances that still have type parameters in them after
		// substitution. This occurs when a type is defined while nested
		// in a generic context and is not fully instantiated yet.
		// We need to wait until we find a full instantiation of the type.
		return
	}

	c.instances.Add(Instance{
		Object: obj,
		TArgs:  tArgs,
		TNest:  tNest,
	})

	if t, ok := obj.Type().(*types.Named); ok {
		for i := 0; i < t.NumMethods(); i++ {
			method := t.Method(i)
			c.instances.Add(Instance{
				Object: method.Origin(),
				TArgs:  tArgs,
				TNest:  tNest,
			})
		}
	}
}

// seedVisitor implements ast.Visitor that collects information necessary to
// kickstart generic instantiation discovery.
//
// It serves double duty:
//   - Builds a map from types.Object instances representing generic types,
//     methods and functions to AST nodes that define them.
//   - Collects an initial set of generic instantiations in the non-generic code.
type seedVisitor struct {
	visitor
	objMap  map[types.Object]ast.Node
	mapOnly bool // Only build up objMap, ignore any instances.
}

var _ ast.Visitor = &seedVisitor{}

func (c *seedVisitor) Visit(n ast.Node) ast.Visitor {
	// Generic functions, methods and types require type arguments to scan for
	// generic instantiations, remember their node for later and do not descend
	// further.
	switch n := n.(type) {
	case *ast.FuncDecl:
		obj := c.info.Defs[n.Name]
		sig := obj.Type().(*types.Signature)
		if sig.TypeParams().Len() != 0 || sig.RecvTypeParams().Len() != 0 {
			c.objMap[obj] = n
			return &seedVisitor{
				visitor: c.visitor,
				objMap:  c.objMap,
				mapOnly: true,
			}
		}
	case *ast.TypeSpec:
		obj := c.info.Defs[n.Name]
		named, ok := obj.Type().(*types.Named)
		if !ok {
			break
		}
		if named.TypeParams().Len() != 0 && named.TypeArgs().Len() == 0 {
			c.objMap[obj] = n
			return nil
		}
	}

	if !c.mapOnly {
		// Otherwise check for fully defined instantiations and descend further into
		// the AST tree.
		c.visitor.Visit(n)
	}
	return c
}

// Collector scans type-checked AST tree and adds discovered generic type and
// function instances to the InstanceSet.
//
// Collector will scan non-generic code for any instantiations of generic types
// or functions and add them to the InstanceSet. Then it will scan generic types
// and function with discovered sets of type arguments for more instantiations,
// until no new ones are discovered.
//
// InstanceSet may contain unprocessed instances of generic types and functions,
// which will be also scanned, for example found in depending packages.
//
// Note that instances of generic type methods are automatically added to the
// set whenever their receiver type instance is encountered.
type Collector struct {
	TContext  *types.Context
	Info      *types.Info
	Instances *PackageInstanceSets
}

// Scan package files for generic instances.
func (c *Collector) Scan(pkg *types.Package, files ...*ast.File) {
	if c.Info.Instances == nil || c.Info.Defs == nil {
		panic(fmt.Errorf("types.Info must have Instances and Defs populated"))
	}
	objMap := map[types.Object]ast.Node{}

	// Collect instances of generic objects in non-generic code in the package and
	// add then to the existing InstanceSet.
	sc := seedVisitor{
		visitor: visitor{
			instances: c.Instances,
			resolver:  nil,
			info:      c.Info,
		},
		objMap: objMap,
	}
	for _, file := range files {
		ast.Walk(&sc, file)
	}

	for iset := c.Instances.Pkg(pkg); !iset.exhausted(); {
		inst, _ := iset.next()

		switch typ := inst.Object.Type().(type) {
		case *types.Signature:
			c.scanSignature(inst, typ, objMap)

		case *types.Named:
			c.scanNamed(inst, typ, objMap)
		}
	}
}

func (c *Collector) scanSignature(inst Instance, typ *types.Signature, objMap map[types.Object]ast.Node) {
	v := visitor{
		instances: c.Instances,
		resolver:  NewResolver(c.TContext, inst),
		info:      c.Info,
		tNest:     inst.TArgs,
	}
	ast.Walk(&v, objMap[inst.Object])
}

func (c *Collector) scanNamed(inst Instance, typ *types.Named, objMap map[types.Object]ast.Node) {
	obj := typ.Obj()
	node := objMap[obj]
	if node == nil {
		// Types without an entry in objMap are concrete types
		// that are defined in a generic context. Skip them.
		return
	}

	v := visitor{
		instances: c.Instances,
		resolver:  NewResolver(c.TContext, inst),
		info:      c.Info,
		tNest:     inst.TNest,
	}
	ast.Walk(&v, node)
}
