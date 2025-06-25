package grouper

import (
	"fmt"
	"go/types"

	"github.com/gopherjs/gopherjs/compiler/internal/sequencer"
)

type Decl interface {
	Grouper() *Info
	comparable
}

// Group groups the declarations by their dependencies and set the group number
// (i.e. the dependency depth) for each Info.
//
// This returns the group count, where each decl with the same
// group number is at the same depth and can be initialized together.
// All group numbers will be in the range [0, count).
//
// This assumes that the `Grouper() *Info` methods on the declarations will
// consistently return the same unique *Info for each declaration.
//
// This may panic with ErrCycleDetected if a cycle is detected in the dependency
// graph created by the declarations' types. (see [Sequencer] for more details)
//
// [Sequencer]: ../sequencer/sequencer.go
func Group[D Decl](decl map[D]struct{}) int {
	g := prepareGrouper(decl)
	for d := range decl {
		g.assignGroup(d)
	}
	return g.count()
}

// ToMermaid generates a mermaid diagram string for the given declarations.
// This is useful for visualizing the dependency graph of the declarations
// any any possible cycles while debugging the type initialization order.
//
// This will not panic if a cycle is detected in the dependency graph,
// instead it will indicate the declarations involved in the cycle with red
// but the depth groups may be incorrect.
//
// The `toString` function is used to convert the declaration to a string
// for the mermaid diagram. If `toString` is nil, then `%v` is used.
func ToMermaid[D Decl](decl map[D]struct{}, toString func(d D) string) string {
	g := prepareGrouper(decl)
	return g.toMermaid(decl, toString)
}

func prepareGrouper[D Decl](decl map[D]struct{}) *grouper[D] {
	g := &grouper[D]{
		typeMap: make(map[types.Type][]*Info, len(decl)),
		seq:     sequencer.New[*Info](),
	}
	for d := range decl {
		g.addDecl(d)
	}
	for d := range decl {
		g.addDeps(d)
	}
	return g
}

type grouper[D Decl] struct {
	typeMap map[types.Type][]*Info
	seq     sequencer.Sequencer[*Info]
}

func (g *grouper[D]) addDecl(d D) {
	info := d.Grouper()
	if info == nil || info.typ == nil {
		// If the decl has no type set, then it was a type
		// that doesn't needed to be ordered, so we can skip it.
		info.Group = 0
		return
	}
	g.typeMap[info.typ] = append(g.typeMap[info.typ], info)
	g.seq.Add(info)
}

func (g *grouper[D]) addDeps(d D) {
	info := d.Grouper()
	if !g.seq.Has(info) {
		// If the sequencer doesn't have this decl, then it was a type
		// that doesn't needed to be ordered, so we can skip it.
		return
	}

	for dep := range info.dep {
		if depInfos, ok := g.typeMap[dep]; ok {
			g.seq.Add(info, depInfos...)
		} else {
			panic(fmt.Errorf(`missing dependency id for %v from %v`, dep, d))
		}
	}
}

func (g *grouper[D]) count() int {
	return g.seq.DepthCount()
}

func (g *grouper[D]) assignGroup(d D) {
	info := d.Grouper()
	// Calling `Depth` may perform sequencing if it hasn't been run before.
	// It may cause a panic if a cycle is detected,
	// but the cycle might not involve the current declaration and the panic
	// would have occurred with any other declaration too.
	depth := g.seq.Depth(info)
	// If the depth is negative, then decl was not in the sequencer
	// and was already assigned to group 0.
	if depth >= 0 {
		info.Group = depth
	}
}

func (g *grouper[D]) toMermaid(decl map[D]struct{}, toString func(d D) string) string {
	if toString == nil {
		toString = func(d D) string {
			return fmt.Sprintf("%v", d)
		}
	}

	infoMap := make(map[*Info]D, len(decl))
	for d := range decl {
		if info := d.Grouper(); g.seq.Has(info) {
			infoMap[info] = d
		}
	}

	return g.seq.ToMermaid(func(info *Info) string {
		if decl, ok := infoMap[info]; ok {
			return toString(decl)
		}
		// This shouldn't happen, but handle it gracefully anyway.
		return `unknown decl`
	})
}
