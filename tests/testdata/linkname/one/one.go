// Package one is a root of test dependency tree, importing packages two and
// three. It ensures a deterministic import and initialization order of the
// test packages.
package one

import (
	_ "unsafe" // for go:linkname

	"github.com/gopherjs/gopherjs/tests/testdata/linkname/three"
	"github.com/gopherjs/gopherjs/tests/testdata/linkname/two"
)

// DoOne is a regular function from the package one to demonstrate a call
// without any special linking trickery.
func DoOne() string {
	return "doing one"
}

// doInternalOne is a function implemented in package one, but actually called
// by package two using a go:linkname directive to gain access to it. Note:
// dead-code elimination must be able to preserve this function.
//
// This is a demonstration that an imported package can linkname a function
// from an importer package.
func doInternalOne() string {
	return "doing internal one: " + oneSecret
}

// oneSecret is an unexported variable in the package one, which doInternalOne()
// must be able to access even when called from another package using a linkname
// mechanism.
var oneSecret = "one secret"

// doInternalThree is implemented in the package three, but not exported (for
// example, to not make it a public API), which package one gains access to
// via a go:linkname directive.
//
// This is a demonstration that an importer package can linkname a non-exported
// function from an imported package.
//
//go:linkname doInternalThree github.com/gopherjs/gopherjs/tests/testdata/linkname/three.doInternalThree
func doInternalThree() string

func DoAll() string {
	result := "" +
		DoOne() + "\n" + // Normal function call in the same package.
		two.DoTwo() + "\n" + // Normal cross-package function call.
		two.DoImportedOne() + "\n" + // Call a function that package two linknamed.
		three.DoThree() + "\n" + // Normal cross-package function call.
		"doing imported three: " + doInternalThree() + "\n" // Call a function from another package this package linknamed.
	return result
}
