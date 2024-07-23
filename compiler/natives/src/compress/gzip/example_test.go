//go:build js && wasm
// +build js,wasm

package gzip_test

import (
	"fmt"
)

// The test relies on a local HTTP server, which is not supported under NodeJS.
func Example_compressingReader() {
	fmt.Println("the data to be compressed")
}
