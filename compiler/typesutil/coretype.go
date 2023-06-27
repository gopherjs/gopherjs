package typesutil

import (
	"go/types"
	_ "unsafe" // for go:linkname
)

// Currently go/types doesn't offer a public API to determine type's core type.
// Instead of importing a third-party reimplementation, I opted to hook into
// the unexported implementation go/types already has.
//
// If https://github.com/golang/go/issues/60994 gets accepted, we will be able
// to switch to the official API.

// CoreType of the given type, or nil of it has no core type.
// https://go.dev/ref/spec#Core_types
func CoreType(t types.Type) types.Type { return coreTypeImpl(t) }

//go:linkname coreTypeImpl go/types.coreType
func coreTypeImpl(t types.Type) types.Type
