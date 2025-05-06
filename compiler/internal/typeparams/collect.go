package typeparams

import (
	"fmt"
	"go/ast"
	"go/types"

	"github.com/gopherjs/gopherjs/compiler/typesutil"
	"github.com/gopherjs/gopherjs/internal/govendor/subst"
)

// Resolver translates types defined in terms of type parameters into concrete
// types, given a mapping from type params to type arguments.
type Resolver struct {
	subster *subst.Subster
	selMemo map[typesutil.Selection]typesutil.Selection
}

// NewResolver creates a new Resolver with tParams entries mapping to tArgs
// entries with the same index.
func NewResolver(tc *types.Context, tParams *types.TypeParamList, tArgs []types.Type, parent *Resolver) *Resolver {
	var nest *subst.Subster
	if parent != nil {
		nest = parent.subster
	}
	r := &Resolver{
		subster: subst.New(tc, tParams, tArgs, nest),
		selMemo: map[typesutil.Selection]typesutil.Selection{},
	}
	return r
}

// Substitute replaces references to type params in the provided type definition
// with the corresponding concrete types.
func (r *Resolver) Substitute(typ types.Type) types.Type {
	if r == nil || r.subster == nil || typ == nil {
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
	if r == nil || r.subster == nil || sel == nil {
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

func (r *Resolver) String() string {
	if r == nil {
		return `{}`
	}
	return r.subster.String()
}

// visitor implements ast.Visitor and collects instances of generic types and
// functions into an InstanceSet.
//
// When traversing an AST subtree corresponding to a generic type, method or
// function, Resolver must be provided mapping the type parameters into concrete
// types.
//
// seedVisitor implements ast.Visitor that collects information necessary to
// kickstart generic instantiation discovery.
//
// It serves double duty:
//   - Builds a map from types.Object instances representing generic types,
//     methods and functions to AST nodes that define them.
//   - Collects an initial set of generic instantiations in the non-generic code.
type visitor struct {
	instances   *PackageInstanceSets
	resolver    *Resolver
	info        *types.Info
	objMap      map[types.Object]ast.Node
	collectInst bool
	mapObjects  bool
}

var _ ast.Visitor = &visitor{}

func (c *visitor) Visit(n ast.Node) ast.Visitor {
	if c.mapObjects {
		// Generic functions, methods and types require type arguments
		// to scan for generic instantiations, remember their node for
		// later and do not descend further.
		if v, stop := c.collectObjMap(n); stop {
			return v
		}
	}

	if c.collectInst {
		// Check for fully defined instantiations and descend further into
		// the AST tree.
		c.collectInstances(n)
	}

	return c
}

func (c *visitor) collectInstances(n ast.Node) {
	ident, ok := n.(*ast.Ident)
	if !ok {
		return
	}

	instance, ok := c.info.Instances[ident]
	if !ok {
		return
	}

	obj := c.info.ObjectOf(ident)

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
		TArgs:  c.resolver.SubstituteAll(instance.TypeArgs),
	})

	if t, ok := obj.Type().(*types.Named); ok {
		for i := 0; i < t.NumMethods(); i++ {
			method := t.Method(i)
			c.instances.Add(Instance{
				Object: method.Origin(),
				TArgs:  c.resolver.SubstituteAll(instance.TypeArgs),
			})
		}
	}
}

func (c *visitor) collectObjMap(n ast.Node) (ast.Visitor, bool) {
	switch n := n.(type) {
	case *ast.FuncDecl:
		obj := c.info.Defs[n.Name]
		sig := obj.Type().(*types.Signature)
		if sig.TypeParams().Len() != 0 || sig.RecvTypeParams().Len() != 0 {
			c.objMap[obj] = n
			return &visitor{
				instances:   c.instances,
				resolver:    c.resolver,
				info:        c.info,
				objMap:      c.objMap,
				collectInst: false,
				mapObjects:  true,
			}, true
		}

	case *ast.TypeSpec:
		obj := c.info.Defs[n.Name]
		named, ok := obj.Type().(*types.Named)
		if !ok {
			break
		}
		if named.TypeParams().Len() != 0 && named.TypeArgs().Len() == 0 {
			c.objMap[obj] = n
			return nil, true
		}
	}

	return nil, false
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
	sc := visitor{
		instances:   c.Instances,
		resolver:    nil,
		info:        c.Info,
		objMap:      objMap,
		collectInst: true,
		mapObjects:  true,
	}
	for _, file := range files {
		fmt.Printf("[Start] Seed: %s\n", file.Name.Name) // TODO(grantnelson-wf): remove
		ast.Walk(&sc, file)
		fmt.Printf("[Stop] Seed: %s\n\n", file.Name.Name) // TODO(grantnelson-wf): remove
	}

	for iset := c.Instances.Pkg(pkg); !iset.exhausted(); {
		inst, _ := iset.next()
		switch typ := inst.Object.Type().(type) {
		case *types.Signature:
			tParams := SignatureTypeParams(typ)
			v := visitor{
				instances:   c.Instances,
				resolver:    NewResolver(c.TContext, tParams, inst.TArgs, nil),
				info:        c.Info,
				collectInst: true,
				mapObjects:  false,
			}
			fmt.Printf("[Start] Signature: %s\n\t%v\n", inst.TypeString(), v.resolver) // TODO(grantnelson-wf): remove
			fmt.Printf("\t%v\n", tParams)                                              // TODO(grantnelson-wf): remove
			ast.Walk(&v, objMap[inst.Object])
			fmt.Printf("[Stop] Signature: %s\n\n", inst.TypeString()) // TODO(grantnelson-wf): remove
		case *types.Named:
			obj := typ.Obj()
			v := visitor{
				instances:   c.Instances,
				resolver:    NewResolver(c.TContext, typ.TypeParams(), inst.TArgs, nil),
				info:        c.Info,
				collectInst: true,
				mapObjects:  false,
			}
			fmt.Printf("[Start] Named: %s\n", inst.TypeString()) // TODO(grantnelson-wf): remove
			ast.Walk(&v, objMap[obj])
			fmt.Printf("[Stop] Named: %s\n\n", inst.TypeString()) // TODO(grantnelson-wf): remove
		}
	}
}
