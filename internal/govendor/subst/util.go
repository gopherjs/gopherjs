// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package subst

import (
	"go/token"
	"go/types"
)

// assert panics with the mesage msg if p is false.
// Avoid combining with expensive string formatting.
// From https://cs.opensource.google/go/x/tools/+/refs/tags/v0.33.0:go/ssa/util.go;l=28
func assert(p bool, msg string) {
	if !p {
		panic(msg)
	}
}

// From https://cs.opensource.google/go/x/tools/+/refs/tags/v0.33.0:go/ssa/wrappers.go;l=262
func changeRecv(s *types.Signature, recv *types.Var) *types.Signature {
	return types.NewSignatureType(recv, nil, nil, s.Params(), s.Results(), s.Variadic())
}

// declaredWithin reports whether an object is declared within a function.
//
// obj must not be a method or a field.
// From https://cs.opensource.google/go/x/tools/+/refs/tags/v0.33.0:go/ssa/util.go;l=145
func declaredWithin(obj types.Object, fn *types.Func) bool {
	// GOPHERJS: Made `fn` optional so that we can use this on package level types too.
	if fn == nil {
		return false
	}

	if obj.Pos() != token.NoPos {
		return fn.Scope().Contains(obj.Pos()) // trust the positions if they exist.
	}
	if fn.Pkg() != obj.Pkg() {
		return false // fast path for different packages
	}

	// Traverse Parent() scopes for fn.Scope().
	for p := obj.Parent(); p != nil; p = p.Parent() {
		if p == fn.Scope() {
			return true
		}
	}
	return false
}
