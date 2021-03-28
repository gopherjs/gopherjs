// A test program to demonstrate go:linkname directive support.
package main

import "github.com/gopherjs/gopherjs/tests/testdata/linkname/one"

func main() {
	print(one.DoAll())
}
