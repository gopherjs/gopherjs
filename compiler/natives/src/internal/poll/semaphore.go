//go:build js
// +build js

package poll

import (
	_ "unsafe" // For go:linkname
)

//go:linkname runtime_Semacquire sync.runtime_Semacquire
func runtime_Semacquire(s *uint32)

//go:linkname runtime_Semrelease sync.runtime_Semrelease
func runtime_Semrelease(s *uint32)
