// Test of instances of generic types causing dependencies.
// In this tests cat, box, and collections do not import each other directly
// but the main causes instances requiring `collections` to need `box` and
// `box` to need `cat` (thus `collections` indirectly needs `cat`).
// This test is also an attempt at a more realistic scenario.
package main

import (
	"github.com/gopherjs/gopherjs/tests/testdata/gencircle/catbox/box"
	"github.com/gopherjs/gopherjs/tests/testdata/gencircle/catbox/cat"
	"github.com/gopherjs/gopherjs/tests/testdata/gencircle/catbox/collections"
)

func main() {
	s := collections.NewStack[box.Unboxer[cat.Cat]]()
	s.Push(box.Box(cat.Cat{Name: "Erwin"}))
	s.Push(box.Box(cat.Cat{Name: "Dirac"}))
	println(s.Pop().Unbox().Name)
}
