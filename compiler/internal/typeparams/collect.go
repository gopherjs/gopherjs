package typeparams

import (
	"fmt"
	"go/ast"
	"go/types"
)

// visitor implements ast.Visitor and collects instances of generic types and
// functions into an InstanceSet.
//
// When traversing an AST subtree corresponding to a generic type, method or
// function, Resolver must be provided mapping the type parameters into concrete
// types.
type visitor struct {
	instances *PackageInstanceSets
	resolver  *Resolver
	info      *types.Info

	nestTParams *types.TypeParamList // The type parameters for a nested context.
	nestTArgs   []types.Type         // The type arguments for a nested context.
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
	var nestTParams *types.TypeParamList
	var nestTArgs []types.Type
	if obj.Parent().Contains(ident.Pos()) {
		nestTParams = c.nestTParams
		nestTArgs = c.nestTArgs
	}

	c.addInstance(obj, tArgs, nestTParams, nestTArgs)
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

	c.addInstance(obj, nil, c.resolver.TypeParams(), c.resolver.TypeArgs())
}

func (c *visitor) addInstance(obj types.Object, tArgList *types.TypeList, nestTParams *types.TypeParamList, nestTArgs []types.Type) {
	tArgs := c.resolver.SubstituteAll(tArgList)
	if isGeneric(nestTParams, tArgs) {
		// Skip any instances that still have type parameters in them after
		// substitution. This occurs when a type is defined while nested
		// in a generic context and is not fully instantiated yet.
		// We need to wait until we find a full instantiation of the type.
		return
	}

	c.instances.Add(Instance{
		Object: obj,
		TArgs:  tArgs,
		TNest:  nestTArgs,
	})

	if t, ok := obj.Type().(*types.Named); ok {
		for i := 0; i < t.NumMethods(); i++ {
			method := t.Method(i)
			c.instances.Add(Instance{
				Object: method.Origin(),
				TArgs:  tArgs,
				TNest:  nestTArgs,
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
	Instances *PackageInstanceSets

	objMap  map[types.Object]ast.Node
	infoMap map[string]*types.Info
}

// Scan package files for generic instances.
func (c *Collector) Scan(info *types.Info, pkg *types.Package, files ...*ast.File) {
	if c.objMap == nil {
		c.objMap = map[types.Object]ast.Node{}
	}
	if c.infoMap == nil {
		c.infoMap = map[string]*types.Info{}
	}

	// Check the info for this package then record it for later use.
	if info.Instances == nil || info.Defs == nil {
		panic(fmt.Errorf("types.Info must have Instances and Defs populated"))
	}
	c.infoMap[pkg.Path()] = info

	// Collect instances of generic objects in non-generic code in the package and
	// add then to the existing InstanceSet.
	sc := seedVisitor{
		visitor: visitor{
			instances: c.Instances,
			resolver:  nil,
			info:      info,
		},
		objMap: c.objMap,
	}
	for _, file := range files {
		ast.Walk(&sc, file)
	}
}

// Finish will finish the collecting instances by propagating instances of
// generic types and functions found in generic code. The generic code is
// rescanned with in an instances context to find internally defined instances.
//
// This should only be called after all the files are scanned.
func (c *Collector) Finish() {
	for !c.Instances.allExhausted() {
		for pkgPath, instances := range *c.Instances {
			c.propagate(pkgPath, instances)
		}
	}
}

func (c *Collector) propagate(pkgPath string, instances *InstanceSet) {
	info := c.infoMap[pkgPath]
	for iset := instances; !iset.exhausted(); {
		inst, _ := iset.next()

		switch typ := inst.Object.Type().(type) {
		case *types.Signature:
			c.scanSignature(inst, typ, info)

		case *types.Named:
			c.scanNamed(inst, typ, info)
		}
	}
}

func (c *Collector) scanSignature(inst Instance, typ *types.Signature, info *types.Info) {
	v := visitor{
		instances: c.Instances,
		resolver:  NewResolver(c.TContext, inst),
		info:      info,

		nestTParams: SignatureTypeParams(typ),
		nestTArgs:   inst.TArgs,
	}
	ast.Walk(&v, c.objMap[inst.Object])
}

func (c *Collector) scanNamed(inst Instance, typ *types.Named, info *types.Info) {
	obj := typ.Obj()
	node := c.objMap[obj]
	if node == nil {
		// Types without an entry in objMap are concrete types
		// that are defined in a generic context. Skip them.
		return
	}

	var nestTParams *types.TypeParamList
	nest := FindNestingFunc(obj)
	if nest != nil {
		nestTParams = SignatureTypeParams(nest.Type().(*types.Signature))
	}

	v := visitor{
		instances: c.Instances,
		resolver:  NewResolver(c.TContext, inst),
		info:      info,

		nestTParams: nestTParams,
		nestTArgs:   inst.TNest,
	}
	ast.Walk(&v, node)
}
