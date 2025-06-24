package compiler

import (
	"fmt"
	"go/types"

	"github.com/gopherjs/gopherjs/compiler/internal/sequencer"
)

func SetInitGroups(tc *types.Context, decls map[*Decl]struct{}) {
	g := &grouper{
		tc:        tc,
		idDeclMap: make(map[int][]*Decl, len(decls)),
		typeIdMap: make(map[types.Type]int, len(decls)),
		seq:       sequencer.New[int](),
	}
	for d := range decls {
		g.addDecl(d)
	}
	g.addTypeDeps()
	fmt.Println(g.seq.ToMermaid()) // TODO(grantnelson-wf): REMOVE
	g.assignDepths()
}

// grouper is a helper for SetInitGroups
//
// Note: This uses id's instead of types when sequencing since
// "types.Type to satisfy comparable requires go1.20 or later", but the types
// still work as a map key, just not as the type argument in sequencer.
type grouper struct {
	tc        *types.Context
	idDeclMap map[int][]*Decl
	typeIdMap map[types.Type]int
	seq       sequencer.Sequencer[int]
}

func (g *grouper) addDecl(d *Decl) {
	typ := g.getDeclType(d)
	if typ == nil {
		d.InitGroup = 0
		return
	}

	if id, ok := g.typeIdMap[typ]; ok {
		g.idDeclMap[id] = append(g.idDeclMap[id], d)
		return
	}

	id := len(g.idDeclMap)
	g.idDeclMap[id] = []*Decl{d}
	g.typeIdMap[typ] = id
	g.seq.Add(id)
}

func (g *grouper) getDeclType(d *Decl) types.Type {
	inst := d.Instance
	if inst == nil || inst.Object == nil {
		return nil
	}

	switch t := inst.Object.Type().(type) {
	case *types.Named:
		return inst.Resolve(g.tc)
	default:
		return t
	}
}

func (g *grouper) addTypeDeps() {
	for typ, id := range g.typeIdMap {
		deps := make(map[types.Type]struct{})
		for _, d := range g.idDeclMap[id] {
			g.collectDeclDeps(d, deps)
		}

		for dep := range deps {
			if depId, ok := g.typeIdMap[dep]; ok {
				g.seq.Add(id, depId)
			} else {
				panic(fmt.Errorf(`missing dependency id for %v from %v`, dep, typ))
			}
		}
	}
}

func (g *grouper) collectDeclDeps(d *Decl, deps map[types.Type]struct{}) {
	fmt.Printf(">> Adding %s\n", d.FullName) // TODO(grantnelson-wf): REMOVE
	inst := d.Instance

	for _, nestArg := range inst.TNest {
		g.collectDep(nestArg, deps)
	}

	for _, tArg := range inst.TArgs {
		g.collectDep(tArg, deps)
	}

	switch t := inst.Object.Type().(type) {
	case interface{ TypeArgs() *types.TypeList }:
		// Handles *type.Named and *types.Alias (in go1.22)
		for i := t.TypeArgs().Len() - 1; i >= 0; i-- {
			g.collectDep(t.TypeArgs().At(i), deps)
		}

	case *types.Signature:
		if r := t.Recv(); r != nil {
			fmt.Printf(">> Recv: %s in %v\n", r.Type(), inst) // TODO(grantnelson-wf): REMOVE
			//g.collectDep(r.Type(), deps)
		}

	case *types.Map:
		g.collectDep(t.Key(), deps)
		g.collectDep(t.Elem(), deps)

	case interface{ Elem() types.Type }:
		// Handles *types.Pointer, *types.Slice, *types.Array, and *types.Chan
		g.collectDep(t.Elem(), deps)
	}
}

func (g *grouper) collectDep(typ types.Type, deps map[types.Type]struct{}) {
	switch t := typ.(type) {
	case nil, *types.Basic:
		// Nil and Basic types aren't used as dependencies
		// since they don't have unique declarations.
	default:
		deps[t] = struct{}{}
	}
}

func (g *grouper) assignDepths() {
	for depth := g.seq.DepthCount() - 1; depth >= 0; depth-- {
		fmt.Printf(">> Grouping depth: %d\n", depth) // TODO(grantnelson-wf): REMOVE
		group := g.seq.Group(depth)
		for _, id := range group {
			decls := g.idDeclMap[id]
			for _, d := range decls {
				fmt.Printf("\t%s\n\t\t%v\n", d.FullName, d.Instance.Object.Type()) // TODO(grantnelson-wf): REMOVE
				d.InitGroup = depth
			}
		}
	}
}
