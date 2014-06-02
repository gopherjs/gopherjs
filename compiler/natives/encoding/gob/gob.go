// +build js

package gob

import (
	"github.com/gopherjs/gopherjs/js"
	"io"
)

func NewEncoder(w io.Writer) *Encoder {
	js.Global.Call("$notSupported", "encoding/gob")
	panic("unreachable")
}

func NewDecoder(r io.Reader) *Decoder {
	js.Global.Call("$notSupported", "encoding/gob")
	panic("unreachable")
}
