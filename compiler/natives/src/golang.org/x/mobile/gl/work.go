// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build js

package gl

import (
	"github.com/gopherjs/gopherjs/js"
)

const workbufLen = 3

// JSCanvas returns the canvas element associated with the Context.
// This function is specific to gopherjs.
func JSCanvas(ctx Context) *js.Object {
	if ctx3, ok := ctx.(context3); ok {
		return ctx3.canvas
	}
	return ctx.(*context).canvas
}

type context struct {
	*js.Object
	canvas *js.Object
	debug  int32
}

func (ctx *context) callWebGL2Compat(extension, suffix, method string, args ...interface{}) *js.Object {
	if glesVersion == "GL_ES_2_0" {
		ext := ctx.Call("getExtension", extension)
		if ext == nil {
			panic("gl: context does not support extension: " + extension)
		}
		return ext.Call(method+suffix, args...)
	}
	return ctx.Call(method, args...)
}

func (ctx *context) WorkAvailable() <-chan struct{} { return nil }

type context3 struct {
	*context
}

var glesVersion = func() string {
	canvas := js.Global.Get("document").Call("createElement", "canvas")
	ctx := canvas.Call("getContext", "webgl2")
	if ctx == nil {
		return "GL_ES_2_0"
	}
	return "GL_ES_3_0"
}()

// NewContext creates a js WebGL context.
//
// See the Worker interface for more details on how it is used.
func NewContext() (Context, Worker) {
	glctx := &context{canvas: js.Global.Get("document").Call("createElement", "canvas")}
	if glesVersion == "GL_ES_2_0" {
		glctx.Object = glctx.canvas.Call("getContext", "webgl")
		return glctx, glctx
	}
	glctx.Object = glctx.canvas.Call("getContext", "webgl2")
	return context3{glctx}, glctx
}

// Version returns a GL ES version string, either "GL_ES_2_0" or "GL_ES_3_0".
// Future versions of the gl package may return "GL_ES_3_1".
func Version() string {
	return glesVersion
}

func (ctx *context) DoWork() {
	select {}
}
