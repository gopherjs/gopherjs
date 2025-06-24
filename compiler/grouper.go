package compiler

import (
	"fmt"
	"go/types"
	"strings"

	"github.com/gopherjs/gopherjs/compiler/internal/sequencer"
)

func SetInitGroups(tc *types.Context, decls map[*Decl]struct{}) {
	g := &grouper{
		tc:      tc,
		typeMap: make(map[types.Type][]*Decl, len(decls)),
		seq:     sequencer.New[*Decl](),
	}
	for d := range decls {
		g.addDecl(d)
	}
	for d := range decls {
		g.addDeclDeps(d)
	}
	fmt.Println(g.seq.ToMermaid(func(d *Decl) string { // TODO(grantnelson-wf): REMOVE
		return strings.ReplaceAll(d.FullName, `github.com/gopherjs/gopherjs/compiler/`, ``)
	}))
	for d := range decls {
		g.assignGroup(d)
	}
}

// grouper is a helper for SetInitGroups
//
// Note: This uses id's instead of types when sequencing since
// "types.Type to satisfy comparable requires go1.20 or later", but the types
// still work as a map key, just not as the type argument in sequencer.
type grouper struct {
	tc      *types.Context
	typeMap map[types.Type][]*Decl
	seq     sequencer.Sequencer[*Decl]
}

func (g *grouper) addDecl(d *Decl) {
	typ := g.getDeclType(d)
	if typ == nil {
		// If the declaration has no instance or the type isn't one that
		// needs to be ordered, we can skip it.
		d.InitGroup = 0
		return
	}
	g.typeMap[typ] = append(g.typeMap[typ], d)
	g.seq.Add(d)
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

func (g *grouper) addDeclDeps(d *Decl) {
	if !g.seq.Has(d) {
		// If the sequencer doesn't have this decl, then it wasn't a type
		// that needed to be ordered, so we can skip it.
		return
	}

	inst := d.Instance
	for _, nestArg := range inst.TNest {
		g.addDepType(d, nestArg)
	}
	for _, tArg := range inst.TArgs {
		g.addDepType(d, tArg)
	}

	switch t := inst.Object.Type().(type) {
	case interface{ TypeArgs() *types.TypeList }:
		// Handles *type.Named and *types.Alias (in go1.22)
		for i := t.TypeArgs().Len() - 1; i >= 0; i-- {
			g.addDepType(d, t.TypeArgs().At(i))
		}

	case *types.Signature:
		if r := t.Recv(); r != nil {
			//g.collectDep(r.Type(), deps)
		}

	case *types.Map:
		g.addDepType(d, t.Key())
		g.addDepType(d, t.Elem())

	case interface{ Elem() types.Type }:
		// Handles *types.Pointer, *types.Slice, *types.Array, and *types.Chan
		g.addDepType(d, t.Elem())
	}
}

func (g *grouper) addDepType(d *Decl, depTyp types.Type) {
	switch depTyp.(type) {
	case nil, *types.Basic:
		// Nil and Basic types aren't used as dependencies
		// since they don't have unique declarations.
		return
	}

	if depDecls, ok := g.typeMap[depTyp]; ok {
		g.seq.Add(d, depDecls...)
	} else {
		panic(fmt.Errorf(`missing dependency id for %v from %v`, depTyp, d))
	}
}

func (g *grouper) assignGroup(d *Decl) {
	depth := g.seq.Depth(d)
	// If the depth is negative, then decl was not in the sequencer
	// and was already assigned to group 0.
	if depth >= 0 {
		d.InitGroup = depth
	}
}
