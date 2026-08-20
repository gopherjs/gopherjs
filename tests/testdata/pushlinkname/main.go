package main

import (
	"github.com/gopherjs/gopherjs/tests/testdata/pushlinkname/anotherPushLink"
	"github.com/gopherjs/gopherjs/tests/testdata/pushlinkname/pullLink"
	"github.com/gopherjs/gopherjs/tests/testdata/pushlinkname/pushLink"
)

func main() {
	pullLink.Run()
	pushLink.Run()
	anotherPushLink.Run()
}
