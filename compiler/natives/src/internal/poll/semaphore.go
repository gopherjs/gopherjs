//go:build js

package poll

import (
	_ "unsafe" // For go:linkname
)

//go:linkname runtime_Semacquire sync.runtime_Semacquire
//gopherjs:replace
func runtime_Semacquire(s *uint32)

//go:linkname runtime_Semrelease sync.runtime_Semrelease
//gopherjs:replace
func runtime_Semrelease(s *uint32)
