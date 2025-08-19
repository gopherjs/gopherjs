// Test of instances of generic types causing inverted dependencies.
// `bar.Bar[Entity]` requires `foo.Entity` but is imported in `foo`, meaning
// that `bar` is added to the package list prior `foo`. The setup of types
// must allow for `foo` to be added before `bar` types to lookup `foo`.
package main

import (
	"github.com/gopherjs/gopherjs/tests/testdata/gencircle/simple/bar"
	"github.com/gopherjs/gopherjs/tests/testdata/gencircle/simple/foo"
)

func main() {
	e := foo.Entity{
		Ref: bar.Bar[foo.Entity]{
			Next: &foo.Entity{
				Name: `I am Next`,
			},
		},
	}
	println(e.Ref.Next.Name)
}
