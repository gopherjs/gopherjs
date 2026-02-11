//go:build js

package reflectlite

import "internal/abi"

// GOPHERJS: For some reason the original code left mapType and aliased the rest
// to the ABI version. mapType is not used so this is an alias to override the
// left over refactor cruft.
//
//gopherjs:replace
type mapType = abi.MapType

//gopherjs:replace
func (t rtype) Comparable() bool {
	return toAbiType(t).Comparable()
}
