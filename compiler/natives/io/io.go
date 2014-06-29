// +build js

package io

import (
	"runtime"
)

func Pipe() (*PipeReader, *PipeWriter) {
	panic(&runtime.NotSupportedError{"io.Pipe"})
}
