// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package subst

import (
	"go/types"
)

// assert panics with the mesage msg if p is false.
// Avoid combining with expensive string formatting.
// From https://cs.opensource.google/go/x/tools/+/refs/tags/v0.17.0:go/ssa/util.go;l=27
func assert(p bool, msg string) {
	if !p {
		panic(msg)
	}
}

// From https://cs.opensource.google/go/x/tools/+/refs/tags/v0.33.0:go/ssa/wrappers.go;l=262
func changeRecv(s *types.Signature, recv *types.Var) *types.Signature {
	return types.NewSignatureType(recv, nil, nil, s.Params(), s.Results(), s.Variadic())
}
