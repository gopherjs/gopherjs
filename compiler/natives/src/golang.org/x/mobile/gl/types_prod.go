// Copyright 2014 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build js
// +build !gldebug

package gl

import (
	"fmt"

	"github.com/gopherjs/gopherjs/js"
)

// Enum is equivalent to GLenum, and is normally used with one of the
// constants defined in this package.
type Enum uint32

// Types are defined a structs so that in debug mode they can carry
// extra information, such as a string name. See typesdebug.go.

// Attrib identifies the location of a specific attribute variable.
type Attrib struct {
	Value uint
}

// Program identifies a compiled shader program.
type Program struct {
	Value *js.Object
}

// Shader identifies a GLSL shader.
type Shader struct {
	Value *js.Object
}

// Buffer identifies a GL buffer object.
type Buffer struct {
	Value *js.Object
}

// Framebuffer identifies a GL framebuffer.
type Framebuffer struct {
	Value *js.Object
}

// A Renderbuffer is a GL object that holds an image in an internal format.
type Renderbuffer struct {
	Value *js.Object
}

// A Texture identifies a GL texture unit.
type Texture struct {
	Value *js.Object
}

// Uniform identifies the location of a specific uniform variable.
type Uniform struct {
	Value *js.Object
}

// A VertexArray is a GL object that holds vertices in an internal format.
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

func (v Attrib) String() string       { return fmt.Sprintf("Attrib(%d)", v.Value) }
func (v Program) String() string      { return fmt.Sprintf("Program(%v)", v.Value) }
func (v Shader) String() string       { return fmt.Sprintf("Shader(%v)", v.Value) }
func (v Buffer) String() string       { return fmt.Sprintf("Buffer(%v)", v.Value) }
func (v Framebuffer) String() string  { return fmt.Sprintf("Framebuffer(%v)", v.Value) }
func (v Renderbuffer) String() string { return fmt.Sprintf("Renderbuffer(%v)", v.Value) }
func (v Texture) String() string      { return fmt.Sprintf("Texture(%v)", v.Value) }
func (v Uniform) String() string      { return fmt.Sprintf("Uniform(%v)", v.Value) }
func (v VertexArray) String() string  { return fmt.Sprintf("VertexArray(%v)", v.Value) }
