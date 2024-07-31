package compiler

import (
	"go/types"
	"sort"

	"github.com/gopherjs/gopherjs/compiler/typesutil"
)

type dceInfo struct {
	// Symbol's identifier used by the dead-code elimination logic, not including
	// package path. If empty, the symbol is assumed to be alive and will not be
	// eliminated. For methods it is the same as its receiver type identifier.
	objectFilter string
	// The second part of the identified used by dead-code elimination for methods.
	// Empty for other types of symbols.
	methodFilter string

	// List of fully qualified (including package path) DCE symbol identifiers the
	// symbol depends on for dead code elimination purposes.
	deps []string
}

func (d *dceInfo) SetAsAlive() {
	d.objectFilter = ""
	d.methodFilter = ""
}

func (d *dceInfo) SetName(o types.Object) {
	if typesutil.IsMethod(o) {
		recv := typesutil.RecvType(o.Type().(*types.Signature)).Obj()
		d.objectFilter = recv.Name()
		if !o.Exported() {
			d.methodFilter = o.Name() + "~"
		}
	} else {
		d.objectFilter = o.Name()
	}
}

func (d *dceInfo) SetDeps(objectSet map[types.Object]bool) {
	var deps []string
	for o := range objectSet {
		qualifiedName := o.Pkg().Path() + "." + o.Name()
		if typesutil.IsMethod(o) {
			qualifiedName += "~"
		}
		deps = append(deps, qualifiedName)
	}
	sort.Strings(deps)
	d.deps = deps
}

func SelectAliveDecls(pkgs []*Archive, gls goLinknameSet) map[*Decl]struct{} {
	type dceDeclInfo struct {
		decl         *Decl
		objectFilter string
		methodFilter string
	}

	byFilter := make(map[string][]*dceDeclInfo)
	var pendingDecls []*Decl // A queue of live decls to find other live decls.
	for _, pkg := range pkgs {
		for _, d := range pkg.Declarations {
			if d.dce.objectFilter == "" && d.dce.methodFilter == "" {
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
			info := &dceDeclInfo{decl: d}
			if d.dce.objectFilter != "" {
				info.objectFilter = pkg.ImportPath + "." + d.dce.objectFilter
				byFilter[info.objectFilter] = append(byFilter[info.objectFilter], info)
			}
			if d.dce.methodFilter != "" {
				info.methodFilter = pkg.ImportPath + "." + d.dce.methodFilter
				byFilter[info.methodFilter] = append(byFilter[info.methodFilter], info)
			}
		}
	}

	dceSelection := make(map[*Decl]struct{}) // Known live decls.
	for len(pendingDecls) != 0 {
		d := pendingDecls[len(pendingDecls)-1]
		pendingDecls = pendingDecls[:len(pendingDecls)-1]

		dceSelection[d] = struct{}{} // Mark the decl as live.

		// Consider all decls the current one is known to depend on and possible add
		// them to the live queue.
		for _, dep := range d.dce.deps {
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
