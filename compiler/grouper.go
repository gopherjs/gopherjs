package compiler

import (
	"fmt"
	"go/types"

	"github.com/gopherjs/gopherjs/compiler/internal/sequencer"
)

func SetInitGroups(tc *types.Context, alive map[*Decl]struct{}) {
	declMap := make(map[string][]*Decl, len(alive))
	typeMap := make(map[types.Type]string, len(alive))
	index := 0
	for d := range alive {
		inst := d.Instance
		if inst == nil {
			d.InitGroup = 0
			continue
		}

		fmt.Printf(">> Decl: %v\n\tinst: %v\n", d.FullName, inst) // TODO(grantnelson-wf): REMOVE

		typ := inst.Resolve(tc)
		if id, ok := typeMap[typ]; ok {
			declMap[id] = append(declMap[id], d)
		} else {
			id := fmt.Sprintf(`%d:%v`, index, d)
			declMap[id] = []*Decl{d}
			typeMap[typ] = id
			index++
		}
	}

	// Sequence the declarations based on their dependencies.
	// Using id's instead of types since "types.Type to satisfy comparable requires go1.20 or later",
	// the types still work as a map key, just not in the type arg in sequencer.
	seq := sequencer.New[string]()
	for typ, id := range typeMap {
		seq.Add(id)
		deps := getTypeDeps(typ)
		for dep := range deps {
			if depId, ok := typeMap[dep]; ok {
				seq.Add(id, depId)
			} else {
				panic(fmt.Errorf(`missing dependency id for %v from %v`, dep, typ))
			}
		}
	}

	fmt.Println(seq.ToMermaid())

	// Write the groups to the declarations.
	count := seq.DepthCount()
	for depth := 0; depth < count; depth++ {
		group := seq.Group(depth)
		for _, id := range group {
			decls := declMap[id]
			for _, d := range decls {
				d.InitGroup = depth
			}
		}
	}
}

func getTypeDeps(typ types.Type) map[types.Type]struct{} {
	deps := map[types.Type]struct{}{}

	switch t := typ.(type) {
	case interface{ TypeArgs() *types.TypeList }:
		// Handles *type.Named and *types.Alias (in go1.22)
		for i := 0; i < t.TypeArgs().Len(); i++ {
			deps[t.TypeArgs().At(i)] = struct{}{}
		}

	case *types.Map:
		deps[t.Key()] = struct{}{}
		deps[t.Elem()] = struct{}{}

	case interface{ Elem() types.Type }:
		// Handles *types.Pointer, *types.Slice, *types.Array, and *types.Chan
		deps[t.Elem()] = struct{}{}
	}

	return deps
}
