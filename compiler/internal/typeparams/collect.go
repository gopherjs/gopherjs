package typeparams

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"github.com/gopherjs/gopherjs/compiler/typesutil"
)

// Resolver translates types defined in terms of type parameters into concrete
// types, given a mapping from type params to type arguments.
type Resolver struct {
	TContext *types.Context
	Map      map[*types.TypeParam]types.Type

	memo typesutil.Map[types.Type]
}

// NewResolver creates a new Resolver with tParams entries mapping to tArgs
// entries with the same index.
func NewResolver(tc *types.Context, tParams []*types.TypeParam, tArgs []types.Type) *Resolver {
	r := &Resolver{
		TContext: tc,
		Map:      map[*types.TypeParam]types.Type{},
	}
	if len(tParams) != len(tArgs) {
		panic(fmt.Errorf("len(tParams)=%d not equal len(tArgs)=%d", len(tParams), len(tArgs)))
	}
	for i := range tParams {
		r.Map[tParams[i]] = tArgs[i]
	}
	return r
}

// Substitute replaces references to type params in the provided type definition
// with the corresponding concrete types.
func (r *Resolver) Substitute(typ types.Type) types.Type {
	if r == nil || r.Map == nil {
		return typ // No substitutions to be made.
	}
	if concrete := r.memo.At(typ); concrete != nil {
		return concrete
	}
	concrete := goTypesCheckerSubst((*types.Checker)(nil), token.NoPos, typ, substMap(r.Map), r.TContext)
	r.memo.Set(typ, concrete)
	return concrete
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

// ToSlice converts TypeParamList into a slice with the sale order of entries.
func ToSlice(tpl *types.TypeParamList) []*types.TypeParam {
	result := make([]*types.TypeParam, tpl.Len())
	for i := range result {
		result[i] = tpl.At(i)
	}
	return result
}

// visitor implements ast.Visitor and collects instances of generic types and
// functions into an InstanceSet.
//
// When traversing an AST subtree corresponding to a generic type, method or
// function, Resolver must be provided mapping the type parameters into concrete
// types.
type visitor struct {
	instances *InstanceSet
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

	instance, ok := c.info.Instances[ident]
	if !ok {
		return
	}

	c.instances.Add(Instance{
		Object: c.info.ObjectOf(ident),
		TArgs:  c.resolver.SubstituteAll(instance.TypeArgs),
	})
	return
}

// seedVisitor implements ast.Visitor that collects information necessary to
// kickstart generic instantiation discovery.
//
// It serves double duty:
//  - Builds a map from types.Object instances representing generic types,
//    methods and functions to AST nodes that define them.
//  - Collects an initial set of generic instantiations in the non-generic code.
type seedVisitor struct {
	visitor
	objMap map[types.Object]ast.Node
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
			return nil
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

	// Otherwise check for fully defined instantiations and descend further into
	// the AST tree.
	c.visitor.Visit(n)
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
// Note that methods of generic types are never added to the InstanceSet, since
// they can be easily inferred from the receiver type instances.
type Collector struct {
	TContext  *types.Context
	Info      *types.Info
	Instances *InstanceSet
}

// Scan package files for generic instances.
func (c *Collector) Scan(files ...*ast.File) {
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

	for !c.Instances.exhausted() {
		inst, _ := c.Instances.next()
		switch typ := inst.Object.Type().(type) {
		case *types.Signature:
			v := visitor{
				instances: c.Instances,
				resolver:  NewResolver(c.TContext, ToSlice(typ.TypeParams()), inst.TArgs),
				info:      c.Info,
			}
			ast.Walk(&v, objMap[inst.Object])
		case *types.Named:
			obj := typ.Obj()
			v := visitor{
				instances: c.Instances,
				resolver:  NewResolver(c.TContext, ToSlice(typ.TypeParams()), inst.TArgs),
				info:      c.Info,
			}
			ast.Walk(&v, objMap[obj])

			for i := 0; i < typ.NumMethods(); i++ {
				method := typ.Method(i)
				v.resolver = NewResolver(c.TContext, ToSlice(method.Type().(*types.Signature).RecvTypeParams()), inst.TArgs)
				ast.Walk(&v, objMap[method])
			}
		}
	}
}
