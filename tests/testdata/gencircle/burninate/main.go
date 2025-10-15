// Test of instances of generic types inverted dependencies.
// The `burnable` imports `dragons` but the instance of `Trogdor` requires
// `burnable`. This is a simple check that the all packages are loaded before
// the types finish setting up. This is similar to the "simple" gencircle test
// except with generic functions and methods.
package main

import (
	"github.com/gopherjs/gopherjs/tests/testdata/gencircle/burninate/burnable"
	"github.com/gopherjs/gopherjs/tests/testdata/gencircle/burninate/dragon"
)

func main() {
	d := dragon.Trogdor[burnable.Cottages]{}
	b := burnable.Cottages{}
	burnable.Burn(d, b)
}
