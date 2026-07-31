package pullLink

import (
	_ "unsafe"

	"github.com/gopherjs/gopherjs/tests/testdata/pushlinkname/foo"
)

//go:linkname pullSetName github.com/gopherjs/gopherjs/tests/testdata/pushlinkname/foo.setName1
func pullSetName(*foo.Foo, string)

//go:linkname pullGetName github.com/gopherjs/gopherjs/tests/testdata/pushlinkname/foo.getName1
func pullGetName(foo.Foo) string

func Run() {
	f := &foo.Foo{}
	input := "Morse"
	pullSetName(f, input)
	name := pullGetName(*f)
	println("Pull Link: " + name)
}
