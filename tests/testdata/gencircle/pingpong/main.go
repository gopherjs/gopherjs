// Test of instances of generic types inverse dependencies.
// This is designed to test when types from package A is used around a type
// from package B, e.g. A.X[B.Y[A.Z]]. The type interfaces bounce back and
// forth between two packages. This means that A can not be simply
// run before B nor after A. The generics have to handle A needing B and
// B needing A to resolve a instances of generic types.
package main

import (
	"github.com/gopherjs/gopherjs/tests/testdata/gencircle/pingpong/cat"
	"github.com/gopherjs/gopherjs/tests/testdata/gencircle/pingpong/collections"
)

func main() {
	s := collections.HashSet[cat.Cat[collections.BadHasher]]{}
	s.Add(cat.Cat[collections.BadHasher]{Name: "Fluffy"})
	s.Add(cat.Cat[collections.BadHasher]{Name: "Mittens"})
	s.Add(cat.Cat[collections.BadHasher]{Name: "Whiskers"})
	println(s.Count(), "elements")
}
