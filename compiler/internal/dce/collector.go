package dce

import (
	"errors"
)

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
// This will overwrite any previous dependencies collected for the given DCE.
func (c *Collector) CollectDCEDeps(decl Decl, f func()) {
	if c.dce != nil {
		panic(errors.New(`called CollectDCEDeps inside another CollectDCEDeps call`))
	}

	c.dce = decl.Dce()
	defer func() { c.dce = nil }()

	f()
}

// DeclareDCEDep records a declaration that is currently being transpiled.
// If a collection in in progress, this will add the dependency of the
// declaration doing the collection.
func (c *Collector) DeclareDCEDep(decl Decl) {
	if c.dce == nil {
		return // Dependencies are not being collected.
	}
	c.dce.addDep(decl.Dce())
}
