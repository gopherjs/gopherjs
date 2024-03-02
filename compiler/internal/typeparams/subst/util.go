// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package subst

import "go/types"

// This file defines a number of miscellaneous utility functions.

//// Sanity checking utilities

// assert panics with the mesage msg if p is false.
// Avoid combining with expensive string formatting.
func assert(p bool, msg string) {
	if !p {
		panic(msg)
	}
}

func changeRecv(s *types.Signature, recv *types.Var) *types.Signature {
	return types.NewSignatureType(recv, nil, nil, s.Params(), s.Results(), s.Variadic())
}
