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

	// alive indicates if the declaration is marked as alive
	// and will not be eliminated.
	alive bool

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

// String gets a human-readable representation of the DCE info.
func (d *Info) String() string {
	tags := ``
	if d.alive {
		tags += `[alive] `
	}
	if d.unnamed() {
		tags += `[unnamed] `
	}
	fullName := d.importPath + `.` + d.objectFilter
	if len(d.methodFilter) > 0 {
		fullName += `.` + d.methodFilter
	}
	return tags + fullName + ` -> [` + strings.Join(d.deps, `, `) + `]`
}

// unnamed returns true if SetName has not been called for this declaration.
// This indicates that the DCE is not initialized.
func (d *Info) unnamed() bool {
	return d.objectFilter == `` && d.methodFilter == ``
}

// isAlive returns true if the declaration is marked as alive.
//
// Returns true if SetAsAlive was called on this declaration or
// if SetName was not called meaning the DCE is not initialized.
func (d *Info) isAlive() bool {
	return d.alive || d.unnamed()
}

// SetAsAlive marks the declaration as alive, meaning it will not be eliminated.
//
// This should be called by an entry point (like main() or init() functions)
// or a variable initializer which has a side effect, consider it live.
func (d *Info) SetAsAlive() {
	d.alive = true
}

// SetName sets the name used by DCE to represent the declaration
// this DCE info is attached to.
func (d *Info) SetName(o types.Object) {
	if !d.unnamed() {
		panic(fmt.Errorf(`may only set the name once for %s`, d.String()))
	}

	d.importPath = o.Pkg().Path()
	if typesutil.IsMethod(o) {
		recv := typesutil.RecvType(o.Type().(*types.Signature)).Obj()
		d.objectFilter = recv.Name()
		if !o.Exported() {
			d.methodFilter = o.Name() + `~`
		}
	} else {
		d.objectFilter = o.Name()
	}
}

// setDeps sets the declaration dependencies used by DCE
// for the declaration this DCE info is attached to.
// This overwrites any prior set dependencies.
func (d *Info) setDeps(objectSet map[types.Object]struct{}) {
	deps := make([]string, 0, len(objectSet))
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
