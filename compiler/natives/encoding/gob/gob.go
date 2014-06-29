// +build js

package gob

import (
	"io"
	"runtime"
)

func NewEncoder(w io.Writer) *Encoder {
	panic(&runtime.NotSupportedError{"encoding/gob"})
}

func NewDecoder(r io.Reader) *Decoder {
	panic(&runtime.NotSupportedError{"encoding/gob"})
}
