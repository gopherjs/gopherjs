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
// types, given a mapping from type params to type arguments.
type Resolver struct {
	tParams *types.TypeParamList
	tArgs   []types.Type
	parent  *Resolver

	// subster is the substitution helper that will perform the actual
	// substitutions. This maybe nil when there are no substitutions but
	// will still usable when nil.
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

// TypeParams is the list of type parameters that this resolver
// (not any parent) will substitute.
func (r *Resolver) TypeParams() *types.TypeParamList {
	if r == nil {
		return nil
	}
	return r.tParams
}

// TypeArgs is the list of type arguments that this resolver
// (not any parent) will resolve to.
func (r *Resolver) TypeArgs() []types.Type {
	if r == nil {
		return nil
	}
	return r.tArgs
}

// Parent is the resolver for the function or method that this resolver
// is nested in. This may be nil if the context for this resolver is not
// nested in another generic function or method.
func (r *Resolver) Parent() *Resolver {
	if r == nil {
		return nil
	}
	return r.parent
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
	// then we apply the current type arguments.
	// If there are no type arguments then the type is defined at the
	// package level, or in a concrete function or method.
	// Otherwise, the type is in a generic function or method
	// and we need to apply the nested type arguments for the instance of the
	// generic function or method the type is defined in.
	var tNest []types.Type
	if obj.Parent().Contains(ident.Pos()) {
		tNest = c.resolver.TypeArgs()
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
		// Found a generic type or not a named type (e.g. type parameter).
		// Don't add generic types yet because they
		// will be added when we find an instance of them.
		return
	}

	c.addInstance(obj, nil, c.resolver.TypeArgs())
}

func (c *visitor) addInstance(obj types.Object, tArgList *types.TypeList, tNest []types.Type) {
	tArgs := c.resolver.SubstituteAll(tArgList)
	for _, ta := range tArgs {
		if _, ok := ta.(*types.TypeParam); ok {
			// Skip any instances that still have type parameters in them after
			// substitution. This occurs when a type is defined while nested
			// in a generic context and is not fully instantiated yet.
			// We need to wait until we find a full instantiation of the type.
			return
		}
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
			tParams := SignatureTypeParams(typ)
			v := visitor{
				instances: c.Instances,
				resolver:  NewResolver(c.TContext, tParams, inst.TArgs, nil),
				info:      c.Info,
			}
			ast.Walk(&v, objMap[inst.Object])

		case *types.Named:
			obj := typ.Obj()
			node := objMap[obj]
			if node == nil {
				// Types without an entry in objMap are concrete types
				// that are defined in a generic context. Skip them.
				continue
			}

			v := visitor{
				instances: c.Instances,
				resolver:  NewResolver(c.TContext, typ.TypeParams(), inst.TArgs, nil),
				info:      c.Info,
			}
			ast.Walk(&v, node)
		}
	}
}
