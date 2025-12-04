package dce

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"go/types"
	"sort"
	"strings"
)

// Info contains information used by the dead-code elimination (DCE) logic to
// determine whether a declaration is alive or dead.
type Info struct {
	// alive indicates if the declaration is marked as alive
	// and will not be eliminated.
	alive bool

	// objectFilter is the primary DCE name for a declaration.
	// This will be the variable, function, or type identifier.
	// For methods it is the receiver type identifier.
	// If empty, the declaration is assumed to be alive.
	objectFilter string

	// methodFilter is the secondary DCE name for a declaration.
	// This will be empty if objectFilter is empty.
	// This will be set to a qualified method name if the objectFilter
	// can not determine if the declaration is alive on it's own.
	// See ./README.md for more information.
	methodFilter string

	// Set of fully qualified (including package path) DCE symbol
	// and/or method names that this DCE declaration depends on.
	deps map[string]struct{}
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
	names := []string{}
	if len(d.objectFilter) > 0 {
		names = append(names, d.objectFilter+` `)
	}
	if len(d.methodFilter) > 0 {
		names = append(names, d.methodFilter+` `)
	}
	return tags + strings.Join(names, `& `) + `-> [` + strings.Join(d.getDeps(), `, `) + `]`
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
//
// The given optional type arguments are used to when the object is a
// function with type parameters or anytime the object doesn't carry them.
// If not given, this attempts to get the type arguments from the object.
func (d *Info) SetName(o types.Object, tNest, tArgs []types.Type) {
	if !d.unnamed() {
		panic(fmt.Errorf(`may only set the name once for %s`, d.String()))
	}

	// Determine name(s) for DCE.
	d.objectFilter, d.methodFilter = getFilters(o, tNest, tArgs)
}

// addDep add a declaration dependencies used by DCE
// for the declaration this DCE info is attached to.
func (d *Info) addDep(o types.Object, tNest, tArgs []types.Type) {
	objectFilter, methodFilter := getFilters(o, tNest, tArgs)
	d.addDepName(objectFilter)
	d.addDepName(methodFilter)
}

// addDepName adds a declaration dependency by name.
func (d *Info) addDepName(depName string) {
	if len(depName) > 0 {
		if d.deps == nil {
			d.deps = make(map[string]struct{})
		}
		d.deps[depName] = struct{}{}
	}
}

// getDeps gets the dependencies for the declaration sorted by name.
func (id *Info) getDeps() []string {
	deps := make([]string, len(id.deps))
	i := 0
	for dep := range id.deps {
		deps[i] = dep
		i++
	}
	sort.Strings(deps)
	return deps
}

type serializableInfo struct {
	Alive        bool
	ObjectFilter string
	MethodFilter string
	Deps         []string
}

func (id *Info) GobEncode() ([]byte, error) {
	buf := &bytes.Buffer{}
	err := id.Write(gob.NewEncoder(buf).Encode)
	return buf.Bytes(), err
}

func (id *Info) MarshalJSON() ([]byte, error) {
	buf := &bytes.Buffer{}
	err := id.Write(json.NewEncoder(buf).Encode)
	return buf.Bytes(), err
}

func (id *Info) GobDecode(data []byte) error {
	return id.Read(gob.NewDecoder(bytes.NewReader(data)).Decode)
}

func (id *Info) UnmarshalJSON(data []byte) error {
	return id.Read(json.NewDecoder(bytes.NewReader(data)).Decode)
}

func (id *Info) Write(encode func(any) error) error {
	si := serializableInfo{
		Alive:        id.alive,
		ObjectFilter: id.objectFilter,
		MethodFilter: id.methodFilter,
		Deps:         id.getDeps(),
	}
	return encode(si)
}

func (id *Info) Read(decode func(any) error) error {
	var si serializableInfo
	if err := decode(&si); err != nil {
		return err
	}

	id.alive = si.Alive
	id.objectFilter = si.ObjectFilter
	id.methodFilter = si.MethodFilter
	id.deps = make(map[string]struct{}, len(si.Deps))
	for _, dep := range si.Deps {
		id.deps[dep] = struct{}{}
	}
	return nil
}
