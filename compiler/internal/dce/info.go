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

	// obj is the Go object the declaration this DCE is for.
	obj types.Object

	// deps is the set of Go objects that this DCE depends on.
	deps map[types.Object]struct{}
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
	objectName, methodName := d.getInfoNames()
	fullName := objectName
	if len(methodName) > 0 {
		objectName += ` &` + methodName
	}
	depNames := `[` + strings.Join(d.getDepNames(), `, `) + `]`
	return tags + fullName + ` -> ` + depNames
}

// unnamed returns true if SetName has not been called for this declaration.
// This indicates that the DCE is not initialized.
func (d *Info) unnamed() bool {
	return d.obj == nil
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
	d.obj = o
}

// addDep adds a declaration dependency for the declaration this
// DCE info is attached to.
func (d *Info) addDep(dep types.Object) {
	d.deps[dep] = struct{}{}
}

func (d *Info) getInfoNames() (objectFilter, methodFilter string) {
	o := d.obj
	importPath := o.Pkg().Path()
	if typesutil.IsMethod(o) {
		recv := typesutil.RecvType(o.Type().(*types.Signature)).Obj()
		objectFilter = importPath + `.` + recv.Name()
		if !o.Exported() {
			methodFilter = importPath + `.` + o.Name() + `~`
		}
	} else {
		objectFilter = importPath + `.` + o.Name()
	}
	return
}

func (d *Info) getDepNames() []string {
	depNames := make([]string, 0, len(d.deps))
	for o := range d.deps {
		qualifiedName := o.Pkg().Path() + "." + o.Name()
		if typesutil.IsMethod(o) {
			qualifiedName += "~"
		}
		depNames = append(depNames, qualifiedName)
	}
	sort.Strings(depNames)
	return depNames
}
