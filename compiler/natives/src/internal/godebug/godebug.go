//go:build js

package godebug

import _ "unsafe" // go:linkname

//go:linkname setUpdate runtime.godebug_setUpdate
func setUpdate(update func(def, env string))
