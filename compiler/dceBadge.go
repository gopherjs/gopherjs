package compiler

type DceBadge struct {
	// Symbol's identifier used by the dead-code elimination logic, not including
	// package path. If empty, the symbol is assumed to be alive and will not be
	// eliminated. For methods it is the same as its receiver type identifier.
	ObjectFilter string
	// The second part of the identified used by dead-code elimination for methods.
	// Empty for other types of symbols.
	MethodFilter string

	// List of fully qualified (including package path) DCE symbol identifiers the
	// symbol depends on for dead code elimination purposes.
	Deps []string
}

type dceInfo struct {
	decl         *Decl
	objectFilter string
	methodFilter string
}

func SelectAliveDecls(pkgs []*Archive, gls goLinknameSet) map[*Decl]struct{} {
	byFilter := make(map[string][]*dceInfo)
	var pendingDecls []*Decl // A queue of live decls to find other live decls.
	for _, pkg := range pkgs {
		for _, d := range pkg.Declarations {
			if d.Dce.ObjectFilter == "" && d.Dce.MethodFilter == "" {
				// This is an entry point (like main() or init() functions) or a variable
				// initializer which has a side effect, consider it live.
				pendingDecls = append(pendingDecls, d)
				continue
			}
			if gls.IsImplementation(d.LinkingName) {
				// If a decl is referenced by a go:linkname directive, we just assume
				// it's not dead.
				// TODO(nevkontakte): This is a safe, but imprecise assumption. We should
				// try and trace whether the referencing functions are actually live.
				pendingDecls = append(pendingDecls, d)
			}
			info := &dceInfo{decl: d}
			if d.Dce.ObjectFilter != "" {
				info.objectFilter = pkg.ImportPath + "." + d.Dce.ObjectFilter
				byFilter[info.objectFilter] = append(byFilter[info.objectFilter], info)
			}
			if d.Dce.MethodFilter != "" {
				info.methodFilter = pkg.ImportPath + "." + d.Dce.MethodFilter
				byFilter[info.methodFilter] = append(byFilter[info.methodFilter], info)
			}
		}
	}

	/*
		keys := make([]string, 0, len(byFilter)) // TODO(gn): REMOVE
		for k := range byFilter {
			if strings.HasPrefix(k, "main") {
				keys = append(keys, k)
			}
		}
		sort.Strings(keys) // TODO(gn): REMOVE
		fmt.Println("byFilter:")
		for _, k := range keys { // TODO(gn): REMOVE
			fmt.Printf("\t%q\n", k) // TODO(gn): REMOVE
		}
	*/

	dceSelection := make(map[*Decl]struct{}) // Known live decls.
	for len(pendingDecls) != 0 {
		d := pendingDecls[len(pendingDecls)-1]
		pendingDecls = pendingDecls[:len(pendingDecls)-1]

		dceSelection[d] = struct{}{} // Mark the decl as live.

		// Consider all decls the current one is known to depend on and possible add
		// them to the live queue.
		for _, dep := range d.Dce.Deps {
			if infos, ok := byFilter[dep]; ok {
				delete(byFilter, dep)
				for _, info := range infos {
					if info.objectFilter == dep {
						info.objectFilter = ""
					}
					if info.methodFilter == dep {
						info.methodFilter = ""
					}
					if info.objectFilter == "" && info.methodFilter == "" {
						pendingDecls = append(pendingDecls, info.decl)
					}
				}
			}
		}
	}
	return dceSelection
}
