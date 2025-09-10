package dce

import (
	"errors"
	"go/types"
)

// Decl is any code declaration that has dead-code elimination (DCE)
// information attached to it.
type Decl interface {
	Dce() *Info
}

// Collector is a tool to collect dependencies for a declaration
// that'll be used in dead-code elimination (DCE).
type Collector struct {
	dce *Info
}

// CollectDCEDeps captures a list of Go objects (types, functions, etc.)
// the code translated inside f() depends on. Then sets those objects
// as dependencies of the given dead-code elimination info.
//
// Only one CollectDCEDeps call can be active at a time.
func (c *Collector) CollectDCEDeps(decl Decl, f func()) {
	if c.dce != nil {
		panic(errors.New(`called CollectDCEDeps inside another CollectDCEDeps call`))
	}

	c.dce = decl.Dce()
	defer func() { c.dce = nil }()

	f()
}

// DeclareDCEDep records that the code that is currently being transpiled
// depends on a given Go object with optional type arguments.
//
// The given optional type arguments are used to when the object is a
// function with type parameters or anytime the object doesn't carry them.
// If not given, this attempts to get the type arguments from the object.
func (c *Collector) DeclareDCEDep(o types.Object, tNest, tArgs []types.Type) {
	if c.dce != nil {
		c.dce.addDep(o, tNest, tArgs)
	}
}
