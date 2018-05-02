// Copyright 2014 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build js
// +build gldebug

package gl

// Alternate versions of the types defined in types.go with extra
// debugging information attached. For documentation, see types.go.

import (
	"fmt"

	"github.com/gopherjs/gopherjs/js"
)

type Enum uint32

type Attrib struct {
	Value uint
	name  string
}

type Program struct {
	Value *js.Object
}

type Shader struct {
	Value *js.Object
}

type Buffer struct {
	Value *js.Object
}

type Framebuffer struct {
	Value *js.Object
}

type Renderbuffer struct {
	Value *js.Object
}

type Texture struct {
	Value *js.Object
}

type Uniform struct {
	Value *js.Object
	name  string
}

type VertexArray struct {
	Value *js.Object
}

func (v Attrib) c() uintptr          { return uintptr(v.Value) }
func (v Enum) c() uintptr            { return uintptr(v) }
func (v Program) c() *js.Object      { return v.Value }
func (v Shader) c() *js.Object       { return v.Value }
func (v Buffer) c() *js.Object       { return v.Value }
func (v Framebuffer) c() *js.Object  { return v.Value }
func (v Renderbuffer) c() *js.Object { return v.Value }
func (v Texture) c() *js.Object      { return v.Value }
func (v Uniform) c() *js.Object      { return v.Value }
func (v VertexArray) c() *js.Object  { return v.Value }

func (v Attrib) String() string       { return fmt.Sprintf("Attrib(%d:%s)", v.Value, v.name) }
func (v Program) String() string      { return fmt.Sprintf("Program(%v)", v.Value) }
func (v Shader) String() string       { return fmt.Sprintf("Shader(%v)", v.Value) }
func (v Buffer) String() string       { return fmt.Sprintf("Buffer(%v)", v.Value) }
func (v Framebuffer) String() string  { return fmt.Sprintf("Framebuffer(%v)", v.Value) }
func (v Renderbuffer) String() string { return fmt.Sprintf("Renderbuffer(%v)", v.Value) }
func (v Texture) String() string      { return fmt.Sprintf("Texture(%v)", v.Value) }
func (v Uniform) String() string      { return fmt.Sprintf("Uniform(%v:%s)", v.Value, v.name) }
func (v VertexArray) String() string  { return fmt.Sprintf("VertexArray(%v)", v.Value) }
