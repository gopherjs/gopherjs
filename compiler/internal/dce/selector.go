package dce

import (
	"fmt"
	"sort"
	"strings"
)

// Decl is any code declaration that has dead-code elimination (DCE)
// information attached to it.
// Since this will be used in a set, it must also be comparable.
type Decl interface {
	Dce() *Info
	comparable
}

// Selector gathers all declarations that are still alive after dead-code elimination.
type Selector[D Decl] struct {
	byFilter map[string][]*declInfo[D]

	// A queue of live decls to find other live decls.
	pendingDecls []D
}

type declInfo[D Decl] struct {
	decl         D
	objectFilter string
	methodFilter string
}

// Include will add a new declaration to be checked as alive or not.
func (s *Selector[D]) Include(decl D, implementsLink bool) {
	if s.byFilter == nil {
		s.byFilter = make(map[string][]*declInfo[D])
	}

	dce := decl.Dce()

	if dce.uninitialized() { // TOD(gn): Remove
		fmt.Printf("dce: declaration is uninitialized: %#v\n", decl)
	}

	if dce.isAlive() {
		s.pendingDecls = append(s.pendingDecls, decl)
		return
	}

	if implementsLink {
		s.pendingDecls = append(s.pendingDecls, decl)
	}

	info := &declInfo[D]{decl: decl}

	if dce.objectFilter != "" {
		info.objectFilter = dce.importPath + "." + dce.objectFilter
		s.byFilter[info.objectFilter] = append(s.byFilter[info.objectFilter], info)
	}

	if dce.methodFilter != "" {
		info.methodFilter = dce.importPath + "." + dce.methodFilter
		s.byFilter[info.methodFilter] = append(s.byFilter[info.methodFilter], info)
	}
}

func (s *Selector[D]) popPending() D {
	max := len(s.pendingDecls) - 1
	d := s.pendingDecls[max]
	s.pendingDecls = s.pendingDecls[:max]
	return d
}

// AliveDecls returns a set of declarations that are still alive
// after dead-code elimination.
// This should only be called once all declarations have been included.
func (s *Selector[D]) AliveDecls() map[D]struct{} {
	dceSelection := make(map[D]struct{}) // Known live decls.
	for len(s.pendingDecls) != 0 {
		d := s.popPending()
		dce := d.Dce()

		dceSelection[d] = struct{}{} // Mark the decl as live.

		// Consider all decls the current one is known to depend on and possible add
		// them to the live queue.
		for _, dep := range dce.deps {
			if infos, ok := s.byFilter[dep]; ok {
				delete(s.byFilter, dep)
				for _, info := range infos {
					if info.objectFilter == dep {
						info.objectFilter = ""
					}
					if info.methodFilter == dep {
						info.methodFilter = ""
					}
					if info.objectFilter == "" && info.methodFilter == "" {
						s.pendingDecls = append(s.pendingDecls, info.decl)
					}
				}
			}
		}
	}

	// TODO(gn): Remove
	strs := make([]string, 0, len(dceSelection))
	for d := range dceSelection {
		if len(d.Dce().fullName) > 0 {
			strs = append(strs, d.Dce().String())
		} else {
			strs = append(strs, fmt.Sprintf(`%#v`, d))
		}
	}
	sort.Strings(strs)
	fmt.Println(strings.Join(strs, "\n"))

	return dceSelection
}
