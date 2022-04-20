//go:build js
// +build js

package net

import (
	_ "unsafe" // For go:linkname
)

//go:linkname fastrand runtime.fastrand
func fastrand() uint32
