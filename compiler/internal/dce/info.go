package dce

import (
	"fmt"
	"go/types"
	"sort"
	"strings"

	"github.com/gopherjs/gopherjs/compiler/typesutil"
)

// Info contains information used by the dead-code elimination (DCE) logic to
// determine whether a declaration is alive or dead.
type Info struct {
	alive    bool
	fullName string

	// importPath is the package path of the package the declaration is in.
	importPath string

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

func (d *Info) String() string {
	return fmt.Sprintf(`%s -> [%s]`, d.fullName, strings.Join(d.deps, `, `))
}

func (d *Info) uninitialized() bool {
	return d.objectFilter == "" && d.methodFilter == "" && !d.alive
}

// isAlive returns true if the declaration is marked as alive.
func (d *Info) isAlive() bool {
	return d.alive
}

// SetAsAlive marks the declaration as alive, meaning it will not be eliminated.
// This must be done after the SetName method is called.
//
// This should be called by an entry point (like main() or init() functions)
// or a variable initializer which has a side effect, consider it live.
func (d *Info) SetAsAlive() {
	d.alive = true
	//d.objectFilter = ""
	//d.methodFilter = ""
}

// SetName sets the name used by DCE to represent the declaration
// this DCE info is attached to.
func (d *Info) SetName(o types.Object) {
	d.importPath = o.Pkg().Path()
	if typesutil.IsMethod(o) {
		recv := typesutil.RecvType(o.Type().(*types.Signature)).Obj()
		d.objectFilter = recv.Name()
		if !o.Exported() {
			d.methodFilter = o.Name() + "~"
		}
	} else {
		d.objectFilter = o.Name()
	}

	d.fullName = d.importPath + "." + d.objectFilter
	if len(d.methodFilter) > 0 {
		d.fullName += `.` + d.methodFilter
	}
}

// SetDeps sets the declaration dependencies used by DCE
// for the declaration this DCE info is attached to.
// This overwrites any prior dependencies set so only call it once.
func (d *Info) SetDeps(objectSet map[types.Object]bool) {
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
