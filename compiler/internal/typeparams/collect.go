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
	subster *subst.Subster
	selMemo map[typesutil.Selection]typesutil.Selection
	parent  *Resolver
}

// NewResolver creates a new Resolver with tParams entries mapping to tArgs
// entries with the same index.
func NewResolver(tc *types.Context, tParams *types.TypeParamList, tArgs []types.Type, parent *Resolver) *Resolver {
	r := &Resolver{
		tParams: tParams,
		tArgs:   tArgs,
		subster: subst.New(tc, tParams, tArgs),
		selMemo: map[typesutil.Selection]typesutil.Selection{},
		parent:  parent,
	}
	return r
}

func (r *Resolver) TypeParams() *types.TypeParamList {
	if r == nil {
		return nil
	}
	return r.tParams
}

func (r *Resolver) TypeArgs() []types.Type {
	if r == nil {
		return nil
	}
	return r.tArgs
}

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
	if r.subster != nil {
		typ = r.subster.Type(typ)
	}
	if r.parent != nil {
		typ = r.parent.Substitute(typ)
	}
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

func (c *visitor) Visit(n ast.Node) (w ast.Visitor) {
	w = c // Always traverse the full depth of the AST tree.

	ident, ok := n.(*ast.Ident)
	if !ok {
		return
	}

	if inst, ok := c.info.Instances[ident]; ok {
		// Found instance of a generic type or function.
		obj := c.info.ObjectOf(ident)
		c.addInstance(obj, inst.TypeArgs)
		return
	}

	if len(c.resolver.TypeArgs()) > 0 {
		if obj, ok := c.info.Defs[ident]; ok && isConcreteType(obj) {
			// Found instance of a concrete type defined inside a generic context.
			c.addInstance(obj, nil)
			return
		}
	}

	return
}

// isConcreteType returns true if the object is a non-generic named type.
func isConcreteType(obj types.Object) bool {
	if obj == nil {
		return false
	}
	typ := obj.Type()
	if ptr, ok := typ.(*types.Pointer); ok {
		typ = ptr.Elem()
	}

	t, ok := typ.(*types.Named)
	return ok && t.TypeParams().Len() == 0
}

func (c *visitor) addInstance(obj types.Object, tArgs *types.TypeList) {
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
	c.instances.Add(Instance{
		Object: obj,
		TArgs:  c.resolver.SubstituteAll(tArgs),
		TNest:  c.resolver.TypeArgs(),
	})

	if t, ok := obj.Type().(*types.Named); ok {
		for i := 0; i < t.NumMethods(); i++ {
			method := t.Method(i)
			c.instances.Add(Instance{
				Object: method.Origin(),
				TArgs:  c.resolver.SubstituteAll(tArgs),
				TNest:  c.resolver.TypeArgs(),
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
