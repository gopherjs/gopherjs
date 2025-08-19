package foo

import "github.com/gopherjs/gopherjs/tests/testdata/gencircle/simple/bar"

type Entity struct {
	Ref  bar.Bar[Entity]
	Name string
}
