//go:build js

package net

import _ "unsafe" // For go:linkname

//go:linkname fastrandu runtime.fastrandu
func fastrandu() uint
