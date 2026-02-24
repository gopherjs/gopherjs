//go:build js

package godebug

import _ "unsafe" // go:linkname

//go:linkname setUpdate runtime.godebug_setUpdate
func setUpdate(update func(def, env string))

// GOPHERJS: Changing from a linked function to a no-op since this is to give
// runtime the ability to do `newNonDefaultInc(name)` instead of
// `godebug.New(name).IncNonDefault` but GopherJS's runtime doesn't need that.
//
//gopherjs:replace
func setNewIncNonDefault(newIncNonDefault func(string) func()) {}
