// Copyright 2014 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build js
// +build !gldebug

package gl

// TODO(crawshaw): should functions on specific types become methods? E.g.
//                 func (t Texture) Bind(target Enum)
//                 this seems natural in Go, but moves us slightly
//                 further away from the underlying OpenGL spec.

import "github.com/gopherjs/gopherjs/js"

func (ctx *context) ActiveTexture(texture Enum) {
	ctx.Call("activateTexture", texture.c())
}

func (ctx *context) AttachShader(p Program, s Shader) {
	ctx.Call("attachShader", p.c(), s.c())
}

func (ctx *context) BindAttribLocation(p Program, a Attrib, name string) {
	ctx.Call("bindAttribLocation", p.c(), a.c(), name)
}

func (ctx *context) BindBuffer(target Enum, b Buffer) {
	ctx.Call("bindBuffer", target.c(), b.c())
}

func (ctx *context) BindFramebuffer(target Enum, fb Framebuffer) {
	ctx.Call("bindFramebuffer", target.c(), fb.c())
}

func (ctx *context) BindRenderbuffer(target Enum, rb Renderbuffer) {
	ctx.Call("bindRenderbuffer", target.c(), rb.c())
}

func (ctx *context) BindTexture(target Enum, t Texture) {
	ctx.Call("bindTexture", target.c(), t.c())
}

func (ctx *context) BindVertexArray(va VertexArray) {
	ctx.callWebGL2Compat("OES_vertex_array_object", "OES", "bindVertexArray", va.c())
}

func (ctx *context) BlendColor(red, green, blue, alpha float32) {
	ctx.Call("blendColor", red, green, blue, alpha)
}

func (ctx *context) BlendEquation(mode Enum) {
	ctx.Call("blendEquation", mode.c())
}

func (ctx *context) BlendEquationSeparate(modeRGB, modeAlpha Enum) {
	ctx.Call("blendEquationSeparate", modeRGB.c(), modeAlpha.c())
}

func (ctx *context) BlendFunc(sfactor, dfactor Enum) {
	ctx.Call("blendFunc", sfactor.c(), dfactor.c())
}

func (ctx *context) BlendFuncSeparate(sfactorRGB, dfactorRGB, sfactorAlpha, dfactorAlpha Enum) {
	ctx.Call("blendFuncSeparate", sfactorRGB.c(), dfactorRGB.c(), sfactorAlpha.c(), dfactorAlpha.c())
}

func (ctx *context) BufferData(target Enum, src []byte, usage Enum) {
	ctx.Call("bindData", target.c(), js.NewArrayBuffer(src), usage.c())
}

func (ctx *context) BufferInit(target Enum, size int, usage Enum) {
	ctx.Call("bufferData", target.c(), size, usage.c())
}

func (ctx *context) BufferSubData(target Enum, offset int, data []byte) {
	ctx.Call("bufferSubData", target.c(), offset, js.NewArrayBuffer(data))
}

func (ctx *context) CheckFramebufferStatus(target Enum) Enum {
	return Enum(ctx.Call("checkFramebufferStatus", target.c()).Int())
}

func (ctx *context) Clear(mask Enum) {
	ctx.Call("clear", mask)
}

func (ctx *context) ClearColor(red, green, blue, alpha float32) {
	ctx.Call("clearColor", red, green, blue, alpha)
}

func (ctx *context) ClearDepthf(d float32) {
	ctx.Call("clearDepth", d)
}

func (ctx *context) ClearStencil(s int) {
	ctx.Call("clearStencil", s)
}

func (ctx *context) ColorMask(red, green, blue, alpha bool) {
	ctx.Call("colorMask", red, green, blue, alpha)
}

func (ctx *context) CompileShader(s Shader) {
	ctx.Call("compileShader", s.c())
}

func (ctx *context) CompressedTexImage2D(target Enum, level int, internalformat Enum, width, height, border int, data []byte) {
	ctx.Call("compressedTexImage2D", target.c(), level, internalformat.c(), width, height, border, data)
}

func (ctx *context) CompressedTexSubImage2D(target Enum, level, xoffset, yoffset, width, height int, format Enum, data []byte) {
	ctx.Call("compressedTexSubImage2D", target.c(), level, xoffset, yoffset, width, height, format.c(), data)
}

func (ctx *context) CopyTexImage2D(target Enum, level int, internalformat Enum, x, y, width, height, border int) {
	ctx.Call("copyTexImage2D", target.c(), level, internalformat.c(), x, y, width, height, border)
}

func (ctx *context) CopyTexSubImage2D(target Enum, level, xoffset, yoffset, x, y, width, height int) {
	ctx.Call("copyTexSubImage2D", target.c(), level, xoffset, yoffset, x, y, width, height)
}

func (ctx *context) CreateBuffer() Buffer {
	return Buffer{Value: ctx.Call("createBuffer")}
}

func (ctx *context) CreateFramebuffer() Framebuffer {
	return Framebuffer{Value: ctx.Call("createFramebuffer")}
}

func (ctx *context) CreateProgram() Program {
	return Program{Value: ctx.Call("createProgram")}
}

func (ctx *context) CreateRenderbuffer() Renderbuffer {
	return Renderbuffer{Value: ctx.Call("createRenderbuffer")}
}

func (ctx *context) CreateShader(ty Enum) Shader {
	return Shader{Value: ctx.Call("createShader")}
}

func (ctx *context) CreateTexture() Texture {
	return Texture{Value: ctx.Call("createTexture")}
}

func (ctx *context) CreateVertexArray() VertexArray {
	return VertexArray{Value: ctx.callWebGL2Compat("OES_vertex_array_object", "OES", "createVertexArray")}
}

func (ctx *context) CullFace(mode Enum) {
	ctx.Call("cullFace", mode.c())
}

func (ctx *context) DeleteBuffer(v Buffer) {
	ctx.Call("deleteBuffer", v.c())
}

func (ctx *context) DeleteFramebuffer(v Framebuffer) {
	ctx.Call("deleteFramebuffer", v.c())
}

func (ctx *context) DeleteProgram(p Program) {
	ctx.Call("deleteProgram", p.c())
}

func (ctx *context) DeleteRenderbuffer(v Renderbuffer) {
	ctx.Call("deleteRenderbuffer", v.c())
}

func (ctx *context) DeleteShader(s Shader) {
	ctx.Call("deleteShader", s.c())
}

func (ctx *context) DeleteTexture(v Texture) {
	ctx.Call("deleteTexture", v.c())
}

func (ctx *context) DeleteVertexArray(v VertexArray) {
	ctx.callWebGL2Compat("OES_vertex_array_object", "OES", "deleteVertexArray", v.c())
}

func (ctx *context) DepthFunc(fn Enum) {
	ctx.Call("depthFunc", fn.c())
}

func (ctx *context) DepthMask(flag bool) {
	ctx.Call("depthMask", flag)
}

func (ctx *context) DepthRangef(n, f float32) {
	ctx.Call("depthRange", n, f)
}

func (ctx *context) DetachShader(p Program, s Shader) {
	ctx.Call("detachShader", p.c(), s.c())
}

func (ctx *context) Disable(cap Enum) {
	ctx.Call("disable", cap.c())
}

func (ctx *context) DisableVertexAttribArray(a Attrib) {
	ctx.Call("disableVertexAttribArray", a.c())
}

func (ctx *context) DrawArrays(mode Enum, first, count int) {
	ctx.Call("drawArrays", mode.c(), first, count)
}

func (ctx *context) DrawElements(mode Enum, count int, ty Enum, offset int) {
	ctx.Call("drawElements", mode.c(), count, ty.c(), offset)
}

func (ctx *context) Enable(cap Enum) {
	ctx.Call("enable", cap.c())
}

func (ctx *context) EnableVertexAttribArray(a Attrib) {
	ctx.Call("enableVertexAttribArray", a.c())
}

func (ctx *context) Finish() {
	ctx.Call("finish")
}

func (ctx *context) Flush() {
	ctx.Call("flush")
}

func (ctx *context) FramebufferRenderbuffer(target, attachment, rbTarget Enum, rb Renderbuffer) {
	ctx.Call("framebufferRenderbuffer", target.c(), attachment.c(), rbTarget.c(), rb.c())
}

func (ctx *context) FramebufferTexture2D(target, attachment, texTarget Enum, t Texture, level int) {
	ctx.Call("framebufferTexture2D", target.c(), attachment.c(), texTarget.c(), t.c(), level)
}

func (ctx *context) FrontFace(mode Enum) {
	ctx.Call("frontFace", mode.c())
}

func (ctx *context) GenerateMipmap(target Enum) {
	ctx.Call("generateMipmap", target.c())
}

func (ctx *context) GetActiveAttrib(p Program, index uint32) (name string, size int, ty Enum) {
	info := ctx.Call("getActiveAttrib", p.c(), index)
	return info.Get("name").String(), info.Get("size").Int(), Enum(info.Get("type").Int())
}

func (ctx *context) GetActiveUniform(p Program, index uint32) (name string, size int, ty Enum) {
	info := ctx.Call("getActiveUniform", p.c(), index)
	return info.Get("name").String(), info.Get("size").Int(), Enum(info.Get("type").Int())
}

func (ctx *context) GetAttachedShaders(p Program) []Shader {
	shaders := ctx.Call("getAttachedShaders", p.c())
	wrapped := make([]Shader, shaders.Length())
	for i := range wrapped {
		wrapped[i].Value = shaders.Index(i)
	}
	return wrapped
}

func (ctx *context) GetAttribLocation(p Program, name string) Attrib {
	return Attrib{Value: uint(ctx.Call("getAttribLocation", p.c(), name).Int())}
}

func (ctx *context) GetBooleanv(dst []bool, pname Enum) {
	param := ctx.Call("getParameter", pname.c())
	for i := 0; i < param.Length(); i++ {
		dst[i] = param.Index(i).Bool()
	}
}

func (ctx *context) GetFloatv(dst []float32, pname Enum) {
	param := ctx.Call("getParameter", pname.c())
	for i := 0; i < param.Length(); i++ {
		dst[i] = float32(param.Index(i).Float())
	}
}

func (ctx *context) GetIntegerv(dst []int32, pname Enum) {
	param := ctx.Call("getParameter", pname.c())
	for i := 0; i < param.Length(); i++ {
		dst[i] = int32(param.Index(i).Int())
	}
}

func (ctx *context) GetInteger(pname Enum) int {
	return ctx.Call("getParameter", pname.c()).Int()
}

func (ctx *context) GetBufferParameteri(target, value Enum) int {
	return ctx.Call("getBufferParameter", target.c(), value.c()).Int()
}

func (ctx *context) GetError() Enum {
	return Enum(ctx.Call("getError").Int())
}

func (ctx *context) GetFramebufferAttachmentParameteri(target, attachment, pname Enum) int {
	return ctx.Call("getFramebufferAttachmentParameter", target.c(), attachment.c(), pname.c()).Int()
}

func (ctx *context) GetProgrami(p Program, pname Enum) int {
	return ctx.Call("getProgramParameter", p.c(), pname.c()).Int()
}

func (ctx *context) GetProgramInfoLog(p Program) string {
	return ctx.Call("getProgramInfoLog", p.c()).String()
}

func (ctx *context) GetRenderbufferParameteri(target, pname Enum) int {
	return ctx.Call("getRenderbufferParameter", target.c(), pname.c()).Int()
}

func (ctx *context) GetShaderi(s Shader, pname Enum) int {
	return ctx.Call("getShaderParameter", s.c(), pname.c()).Int()
}

func (ctx *context) GetShaderInfoLog(s Shader) string {
	return ctx.Call("getShaderInfoLog", s.c()).String()
}

func (ctx *context) GetShaderPrecisionFormat(shadertype, precisiontype Enum) (rangeLow, rangeHigh, precision int) {
	prec := ctx.Call("getShaderPrecisionFormat", shadertype.c(), precisiontype.c())
	return prec.Get("rangeMin").Int(), prec.Get("rangeMax").Int(), prec.Get("precision").Int()
}

func (ctx *context) GetShaderSource(s Shader) string {
	return ctx.Call("getShaderSource", s.c()).String()
}

func (ctx *context) GetString(pname Enum) string {
	return ctx.Call("getParameter", pname.c()).String()
}

func (ctx *context) GetTexParameterfv(dst []float32, target, pname Enum) {
	param := ctx.Call("getTexParameter", target.c(), pname.c())
	for i := 0; i < param.Length(); i++ {
		dst[i] = float32(param.Index(i).Float())
	}
}

func (ctx *context) GetTexParameteriv(dst []int32, target, pname Enum) {
	param := ctx.Call("getTexParameter", target.c(), pname.c())
	for i := 0; i < param.Length(); i++ {
		dst[i] = int32(param.Index(i).Int())
	}
}

func (ctx *context) GetUniformfv(dst []float32, src Uniform, p Program) {
	param := ctx.Call("getUniform", p.c(), src.c())
	for i := 0; i < param.Length(); i++ {
		dst[i] = float32(param.Index(i).Float())
	}
}

func (ctx *context) GetUniformiv(dst []int32, src Uniform, p Program) {
	param := ctx.Call("getUniform", p.c(), src.c())
	for i := 0; i < param.Length(); i++ {
		dst[i] = int32(param.Index(i).Int())
	}
}

func (ctx *context) GetUniformLocation(p Program, name string) Uniform {
	return Uniform{Value: ctx.Call("getUniformLocation", p.c(), name)}
}

func (ctx *context) GetVertexAttribf(src Attrib, pname Enum) float32 {
	var params [1]float32
	ctx.GetVertexAttribfv(params[:], src, pname)
	return params[0]
}

func (ctx *context) GetVertexAttribfv(dst []float32, src Attrib, pname Enum) {
	param := ctx.Call("getVertexAttrib", src.c(), pname.c())
	if param.Get("length") == js.Undefined {
		dst[0] = float32(param.Float())
		return
	}
	for i := 0; i < param.Length(); i++ {
		dst[i] = float32(param.Index(i).Float())
	}
}

func (ctx *context) GetVertexAttribi(src Attrib, pname Enum) int32 {
	var params [1]int32
	ctx.GetVertexAttribiv(params[:], src, pname)
	return params[0]
}

func (ctx *context) GetVertexAttribiv(dst []int32, src Attrib, pname Enum) {
	param := ctx.Call("getVertexAttrib", src.c(), pname.c())
	if param.Get("length") == js.Undefined {
		dst[0] = int32(param.Int())
		return
	}
	for i := 0; i < param.Length(); i++ {
		dst[i] = int32(param.Index(i).Int())
	}
}

func (ctx *context) Hint(target, mode Enum) {
	ctx.Call("hint", target.c(), mode.c())
}

func (ctx *context) IsBuffer(b Buffer) bool {
	return b.Value != nil && ctx.Call("isBuffer", b.c()).Bool()
}

func (ctx *context) IsEnabled(cap Enum) bool {
	return ctx.Call("isEnabled", cap.c()).Bool()
}

func (ctx *context) IsFramebuffer(fb Framebuffer) bool {
	return fb.Value != nil && ctx.Call("isFramebuffer", fb.c()).Bool()
}

func (ctx *context) IsProgram(p Program) bool {
	return p.Value != nil && ctx.Call("isProgram", p.c()).Bool()
}

func (ctx *context) IsRenderbuffer(rb Renderbuffer) bool {
	return rb.Value != nil && ctx.Call("isRenderbuffer", rb.c()).Bool()
}

func (ctx *context) IsShader(s Shader) bool {
	return s.Value != nil && ctx.Call("isShader", s.c()).Bool()
}

func (ctx *context) IsTexture(t Texture) bool {
	return t.Value != nil && ctx.Call("isTexture", t.c()).Bool()
}

func (ctx *context) LineWidth(width float32) {
	ctx.Call("lineWidth", width)
}

func (ctx *context) LinkProgram(p Program) {
	ctx.Call("linkProgram", p.c())
}

func (ctx *context) PixelStorei(pname Enum, param int32) {
	ctx.Call("pixelStorei", pname.c(), param)
}

func (ctx *context) PolygonOffset(factor, units float32) {
	ctx.Call("polygonOffset", factor, units)
}

func (ctx *context) ReadPixels(dst []byte, x, y, width, height int, format, ty Enum) {
	dstView := js.NewArrayBuffer(dst)
	switch ty {
	case UNSIGNED_BYTE:
		dstView = js.Global.Get("Uint8Array").New(dstView)
	case UNSIGNED_SHORT_5_6_5, UNSIGNED_SHORT_4_4_4_4, UNSIGNED_SHORT_5_5_5_1:
		dstView = js.Global.Get("Uint16Array").New(dstView)
	case FLOAT:
		dstView = js.Global.Get("Float32Array").New(dstView)
	}
	ctx.Call("readPixels", x, y, width, height, format.c(), ty.c(), dstView)
}

func (ctx *context) ReleaseShaderCompiler() {
	// no such thing in WebGL
}

func (ctx *context) RenderbufferStorage(target, internalFormat Enum, width, height int) {
	ctx.Call("renderbufferStorage", target.c(), internalFormat.c(), width, height)
}

func (ctx *context) SampleCoverage(value float32, invert bool) {
	ctx.Call("sampleCoverage", value, invert)
}

func (ctx *context) Scissor(x, y, width, height int32) {
	ctx.Call("scissor", x, y, width, height)
}

func (ctx *context) ShaderSource(s Shader, src string) {
	ctx.Call("shaderSource", s.c(), src)
}

func (ctx *context) StencilFunc(fn Enum, ref int, mask uint32) {
	ctx.Call("stencilFunc", fn.c(), ref, mask)
}

func (ctx *context) StencilFuncSeparate(face, fn Enum, ref int, mask uint32) {
	ctx.Call("stencilFuncSeparate", face.c(), fn.c(), ref, mask)
}

func (ctx *context) StencilMask(mask uint32) {
	ctx.Call("stencilMask", mask)
}

func (ctx *context) StencilMaskSeparate(face Enum, mask uint32) {
	ctx.Call("stencilMaskSeparate", face.c(), mask)
}

func (ctx *context) StencilOp(fail, zfail, zpass Enum) {
	ctx.Call("stencilOp", fail.c(), zfail.c(), zpass.c())
}

func (ctx *context) StencilOpSeparate(face, sfail, dpfail, dppass Enum) {
	ctx.Call("stencilOpSeparate", face.c(), sfail.c(), dpfail.c(), dppass.c())
}

func (ctx *context) TexImage2D(target Enum, level int, width, height int, format Enum, ty Enum, data []byte) {
	// It is common to pass TexImage2D a nil data, indicating that a
	// bound GL buffer is being used as the source. In that case, it
	// is not necessary to block.
	var dataView *js.Object
	if len(data) > 0 {
		dataView = js.NewArrayBuffer(data)
		switch ty {
		case UNSIGNED_BYTE:
			dataView = js.Global.Get("Uint8Array").New(dataView)
		case UNSIGNED_SHORT_5_6_5, UNSIGNED_SHORT_4_4_4_4, UNSIGNED_SHORT_5_5_5_1, UNSIGNED_SHORT, HALF_FLOAT:
			dataView = js.Global.Get("Uint16Array").New(dataView)
		case UNSIGNED_INT, UNSIGNED_INT_24_8:
			dataView = js.Global.Get("Uint32Array").New(dataView)
		case FLOAT:
			dataView = js.Global.Get("Float32Array").New(dataView)
		}
	}

	// TODO(crawshaw): GLES3 offset for PIXEL_UNPACK_BUFFER and PIXEL_PACK_BUFFER.
	ctx.Call("texImage2D", target.c(), level, format.c(), width, height, 0, format.c(), ty.c(), dataView)
}

func (ctx *context) TexSubImage2D(target Enum, level int, x, y, width, height int, format, ty Enum, data []byte) {
	dataView := js.NewArrayBuffer(data)
	switch ty {
	case UNSIGNED_BYTE:
		dataView = js.Global.Get("Uint8Array").New(dataView)
	case UNSIGNED_SHORT_5_6_5, UNSIGNED_SHORT_4_4_4_4, UNSIGNED_SHORT_5_5_5_1, HALF_FLOAT:
		dataView = js.Global.Get("Uint16Array").New(dataView)
	case FLOAT:
		dataView = js.Global.Get("Float32Array").New(dataView)
	}
	// TODO(crawshaw): GLES3 offset for PIXEL_UNPACK_BUFFER and PIXEL_PACK_BUFFER.
	ctx.Call("texSubImage2D", target.c(), level, x, y, width, height, format.c(), ty.c(), dataView)
}

func (ctx *context) TexParameterf(target, pname Enum, param float32) {
	ctx.Call("texParameterf", target.c(), pname.c(), param)
}

func (ctx *context) TexParameterfv(target, pname Enum, params []float32) {
	// TODO(BenLubar): no browser currently supports this method
	ctx.Call("texParameterfv", target.c(), pname.c(), params)
}

func (ctx *context) TexParameteri(target, pname Enum, param int) {
	ctx.Call("texParameteri", target.c(), pname.c(), param)
}

func (ctx *context) TexParameteriv(target, pname Enum, params []int32) {
	// TODO(BenLubar): no browser currently supports this method
	ctx.Call("texParameteriv", target.c(), pname.c(), params)
}

func (ctx *context) Uniform1f(dst Uniform, v float32) {
	ctx.Call("uniform1f", dst.c(), v)
}

func (ctx *context) Uniform1fv(dst Uniform, src []float32) {
	ctx.Call("uniform1fv", dst.c(), src)
}

func (ctx *context) Uniform1i(dst Uniform, v int) {
	ctx.Call("uniform1i", dst.c(), v)
}

func (ctx *context) Uniform1iv(dst Uniform, src []int32) {
	ctx.Call("uniform1iv", dst.c(), src)
}

func (ctx *context) Uniform2f(dst Uniform, v0, v1 float32) {
	ctx.Call("uniform2f", dst.c(), v0, v1)
}

func (ctx *context) Uniform2fv(dst Uniform, src []float32) {
	ctx.Call("uniform2fv", dst.c(), src)
}

func (ctx *context) Uniform2i(dst Uniform, v0, v1 int) {
	ctx.Call("uniform2i", dst.c(), v0, v1)
}

func (ctx *context) Uniform2iv(dst Uniform, src []int32) {
	ctx.Call("uniform2iv", dst.c(), src)
}

func (ctx *context) Uniform3f(dst Uniform, v0, v1, v2 float32) {
	ctx.Call("uniform3f", dst.c(), v0, v1, v2)
}

func (ctx *context) Uniform3fv(dst Uniform, src []float32) {
	ctx.Call("uniform3fv", dst.c(), src)
}

func (ctx *context) Uniform3i(dst Uniform, v0, v1, v2 int32) {
	ctx.Call("uniform3i", dst.c(), v0, v1, v2)
}

func (ctx *context) Uniform3iv(dst Uniform, src []int32) {
	ctx.Call("uniform3iv", dst.c(), src)
}

func (ctx *context) Uniform4f(dst Uniform, v0, v1, v2, v3 float32) {
	ctx.Call("uniform4f", dst.c(), v0, v1, v2, v3)
}

func (ctx *context) Uniform4fv(dst Uniform, src []float32) {
	ctx.Call("uniform4fv", dst.c(), src)
}

func (ctx *context) Uniform4i(dst Uniform, v0, v1, v2, v3 int32) {
	ctx.Call("uniform4i", dst.c(), v0, v1, v2, v3)
}

func (ctx *context) Uniform4iv(dst Uniform, src []int32) {
	ctx.Call("uniform4iv", dst.c(), src)
}

func (ctx *context) UniformMatrix2fv(dst Uniform, src []float32) {
	ctx.Call("uniformMatrix2fv", dst.c(), src)
}

func (ctx *context) UniformMatrix3fv(dst Uniform, src []float32) {
	ctx.Call("uniformMatrix3fv", dst.c(), src)
}

func (ctx *context) UniformMatrix4fv(dst Uniform, src []float32) {
	ctx.Call("uniformMatrix4fv", dst.c(), src)
}

func (ctx *context) UseProgram(p Program) {
	ctx.Call("useProgram", p.c())
}

func (ctx *context) ValidateProgram(p Program) {
	ctx.Call("validateProgram", p.c())
}

func (ctx *context) VertexAttrib1f(dst Attrib, x float32) {
	ctx.Call("vertexAttrib1f", dst.c(), x)
}

func (ctx *context) VertexAttrib1fv(dst Attrib, src []float32) {
	ctx.Call("vertexAttrib1fv", dst.c(), src)
}

func (ctx *context) VertexAttrib2f(dst Attrib, x, y float32) {
	ctx.Call("vertexAttrib2f", dst.c(), x, y)
}

func (ctx *context) VertexAttrib2fv(dst Attrib, src []float32) {
	ctx.Call("vertexAttrib2fv", dst.c(), src)
}

func (ctx *context) VertexAttrib3f(dst Attrib, x, y, z float32) {
	ctx.Call("vertexAttrib3f", dst.c(), x, y, z)
}

func (ctx *context) VertexAttrib3fv(dst Attrib, src []float32) {
	ctx.Call("vertexAttrib3fv", dst.c(), src)
}

func (ctx *context) VertexAttrib4f(dst Attrib, x, y, z, w float32) {
	ctx.Call("vertexAttrib4f", dst.c(), x, y, z, w)
}

func (ctx *context) VertexAttrib4fv(dst Attrib, src []float32) {
	ctx.Call("vertexAttrib4fv", dst.c(), src)
}

func (ctx *context) VertexAttribPointer(dst Attrib, size int, ty Enum, normalized bool, stride, offset int) {
	ctx.Call("vertexAttribPointer", dst.c(), size, ty.c(), normalized, stride, offset)
}

func (ctx *context) Viewport(x, y, width, height int) {
	ctx.Call("viewport", x, y, width, height)
}

func (ctx context3) UniformMatrix2x3fv(dst Uniform, src []float32) {
	ctx.Call("uniformMatrix2x3fv", dst.c(), src)
}

func (ctx context3) UniformMatrix3x2fv(dst Uniform, src []float32) {
	ctx.Call("uniformMatrix3x2fv", dst.c(), src)
}

func (ctx context3) UniformMatrix2x4fv(dst Uniform, src []float32) {
	ctx.Call("uniformMatrix2x4fv", dst.c(), src)
}

func (ctx context3) UniformMatrix4x2fv(dst Uniform, src []float32) {
	ctx.Call("uniformMatrix4x2fv", dst.c(), src)
}

func (ctx context3) UniformMatrix3x4fv(dst Uniform, src []float32) {
	ctx.Call("uniformMatrix3x4fv", dst.c(), src)
}

func (ctx context3) UniformMatrix4x3fv(dst Uniform, src []float32) {
	ctx.Call("uniformMatrix4x3fv", dst.c(), src)
}

func (ctx context3) BlitFramebuffer(srcX0, srcY0, srcX1, srcY1, dstX0, dstY0, dstX1, dstY1 int, mask uint, filter Enum) {
	ctx.Call("blitFramebuffer", srcX0, srcY0, srcX1, srcY1, dstX0, dstY0, dstX1, dstY1, mask, filter.c())
}

func (ctx context3) Uniform1ui(dst Uniform, v uint32) {
	ctx.Call("uniform1ui", dst.c(), v)
}

func (ctx context3) Uniform2ui(dst Uniform, v0, v1 uint32) {
	ctx.Call("uniform2ui", dst.c(), v0, v1)
}

func (ctx context3) Uniform3ui(dst Uniform, v0, v1, v2 uint) {
	ctx.Call("uniform3ui", dst.c(), v0, v1, v2)
}

func (ctx context3) Uniform4ui(dst Uniform, v0, v1, v2, v3 uint32) {
	ctx.Call("uniform4ui", dst.c(), v0, v1, v2, v3)
}
