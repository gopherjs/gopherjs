//go:build js

package godebug

import _ "unsafe" // go:linkname

//go:linkname setUpdate runtime.godebug_setUpdate
func setUpdate(update func(def, env string))

//go:linkname setNewIncNonDefault runtime.godebug_setNewIncNonDefault
func setNewIncNonDefault(newIncNonDefault func(string) func())
