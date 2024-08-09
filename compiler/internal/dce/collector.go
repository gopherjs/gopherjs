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
	dependencies map[types.Object]struct{}
}

// CollectDCEDeps captures a list of Go objects (types, functions, etc.)
// the code translated inside f() depends on. Then sets those objects
// as dependencies of the given dead-code elimination info.
//
// Only one CollectDCEDeps call can be active at a time.
// This will overwrite any previous dependencies collected for the given DCE.
func (c *Collector) CollectDCEDeps(decl Decl, f func()) {
	if c.dependencies != nil {
		panic(errors.New(`called CollectDCEDeps inside another CollectDCEDeps call`))
	}

	c.dependencies = make(map[types.Object]struct{})
	defer func() { c.dependencies = nil }()

	f()

	decl.Dce().setDeps(c.dependencies)
}

// DeclareDCEDep records that the code that is currently being transpiled
// depends on a given Go object.
func (c *Collector) DeclareDCEDep(o types.Object) {
	if c.dependencies == nil {
		return // Dependencies are not being collected.
	}
	c.dependencies[o] = struct{}{}
}
