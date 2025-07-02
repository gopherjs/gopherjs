package grouper

import (
	"go/types"
	"sort"
	"strings"

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

// ToGraph generates a Dot diagram string for the given declarations.
// This is useful for visualizing the dependency graph of the declarations
// any any possible cycles while debugging the type initialization order.
//
// This will not panic if a cycle is detected in the dependency graph,
// instead it will indicate the declarations involved in the cycle with red
// but the depth groups may be incorrect.
func ToGraph[D Decl](decl map[D]struct{}, options sequencer.GraphOptions[D]) string {
	g := prepareGrouper(decl)
	return g.toGraph(decl, options)
}

// CyclesToString returns a string representation of the cycles detected in the
// dependency graph of the given declarations.
// This returns an empty string if no cycles are detected.
//
// If `toString` is not nil it is used to convert the declaration to a string
// representation for additional context in the output.
func CyclesToString[D Decl](decl map[D]struct{}, toString func(d D) string) string {
	g := prepareGrouper(decl)
	return g.toCycleString(decl, toString)
}

func prepareGrouper[D Decl](decl map[D]struct{}) *grouper[D] {
	g := &grouper[D]{
		typeMap: map[types.Type]*Info{},
		alias:   map[*Info]*Info{},
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
	typeMap map[types.Type]*Info
	alias   map[*Info]*Info
	seq     sequencer.Sequencer[*Info]
}

func (g *grouper[D]) addDecl(d D) {
	info := d.Grouper()
	if info == nil {
		return
	}
	if info.name == nil && len(info.dep) == 0 {
		// If the decl has no name and no deps, then it was a type
		// that doesn't needed to be ordered, so we can skip it.
		info.Group = 0
		return
	}
	if info.name != nil {
		if rep, has := g.typeMap[info.name]; has {
			// A representative for the named type already exists
			// so this info will alias to it.
			g.alias[info] = rep
		} else {
			// If the name doesn't exist then this info will become
			// the representative for the named type.
			g.typeMap[info.name] = info
			g.seq.Add(info)
		}
	} else {
		// Add any unnamed info to the sequencer.
		g.seq.Add(info)
	}
}

func (g *grouper[D]) addDeps(d D) {
	info := d.Grouper()
	rep := g.unalias(info)
	if !g.seq.Has(rep) {
		// If the sequencer doesn't have this decl, then it was a type
		// that doesn't needed to be ordered, so we can skip it.
		return
	}

	// Add the dependencies from this info to the representative item.
	for dep := range info.dep {
		// If a type can not be found it doesn't exist so isn't initialized.
		// So we can skip adding any dependencies for it.
		if depInfo, ok := g.typeMap[dep]; ok {
			g.seq.Add(rep, depInfo)
		}
	}
}

// unalias returns the representative info for the given info,
// otherwise it returns the given info.
func (g *grouper[D]) unalias(info *Info) *Info {
	if rep, has := g.alias[info]; has {
		return rep
	}
	return info
}

func (g *grouper[D]) getUnaliasMap(decl map[D]struct{}) map[*Info][]D {
	infoMap := make(map[*Info][]D, len(decl))
	for d := range decl {
		if rep := g.unalias(d.Grouper()); g.seq.Has(rep) {
			infoMap[rep] = append(infoMap[rep], d)
		}
	}
	return infoMap
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
	depth := g.seq.Depth(g.unalias(info))
	// If the depth is negative, then decl was not in the sequencer
	// and was already assigned to group 0.
	if depth >= 0 {
		info.Group = depth
	}
}

func (g *grouper[D]) toGraph(decl map[D]struct{}, declOpts sequencer.GraphOptions[D]) string {
	seqOpts := sequencer.GraphOptions[*Info]{
		FilterCycles:              declOpts.FilterCycles,
		StrictFilter:              declOpts.StrictFilter,
		Mermaid:                   declOpts.Mermaid,
		HideGroups:                declOpts.HideGroups,
		LabelItemsWithGroupNumber: declOpts.LabelItemsWithGroupNumber,
	}

	infoMap := g.getUnaliasMap(decl)

	if declOpts.ItemToString != nil {
		seqOpts.ItemToString = func(info *Info) string {
			if decls, ok := infoMap[info]; ok {
				parts := make([]string, len(decls))
				for i, d := range decls {
					parts[i] = declOpts.ItemToString(d)
				}
				return strings.Join(parts, "\n")
			}
			// This shouldn't happen, but handle it gracefully anyway.
			return `unknown decl`
		}
	}

	if declOpts.ItemFilter != nil {
		seqOpts.ItemFilter = func(info *Info) bool {
			if decls, ok := infoMap[info]; ok {
				for _, d := range decls {
					if declOpts.ItemFilter(d) {
						return true
					}
				}
			}
			return false
		}
	}

	return g.seq.ToGraph(seqOpts)
}

func (g *grouper[D]) toCycleString(decl map[D]struct{}, toString func(d D) string) string {
	cycles := g.seq.GetCycles()
	if len(cycles) == 0 {
		return `` // No cycles detected.
	}

	fullToString := func(d D) string {
		if toString != nil {
			return toString(d) + ` ` + d.Grouper().String()
		}
		return d.Grouper().String()
	}

	infoMap := g.getUnaliasMap(decl)
	parts := make([]string, 0, len(cycles))
	for _, cycle := range cycles {
		decl := infoMap[cycle]
		if len(decl) == 1 {
			parts = append(parts, "-- "+fullToString(decl[0]))
			continue
		}

		subParts := make([]string, len(decl))
		for j, d := range decl {
			subParts[j] = fullToString(d)
		}
		sort.Strings(subParts)
		maxIndex := len(subParts) - 1
		part := ",- " + subParts[0] + "\n|  " +
			strings.Join(subParts[1:maxIndex], "\n|  ") +
			"\n`- " + subParts[maxIndex]
		parts = append(parts, part)
	}
	sort.Strings(parts)
	return strings.Join(parts, "\n")
}
