package anotherPushLink

import (
	_ "unsafe"

	"github.com/gopherjs/gopherjs/tests/testdata/pushlinkname/foo"
)

//go:linkname pushSetName
func pushSetName(*foo.Foo, string)

//go:linkname pushGetName
func pushGetName(foo.Foo) string

func Run() {
	f := &foo.Foo{}
	input := "Holland"
	pushSetName(f, input)
	name := pushGetName(*f)
	println("Another Push Link: " + name)
}
