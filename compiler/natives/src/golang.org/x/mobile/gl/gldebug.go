// Copyright 2014 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Code generated from gl.go using go generate. DO NOT EDIT.
// See doc.go for details.

// +build js
// +build gldebug

package gl

import (
	"fmt"
	"log"

	"github.com/gopherjs/gopherjs/js"
)

func (ctx *context) errDrain() string {
	var errs []Enum
	for {
		e := ctx.GetError()
		if e == 0 {
			break
		}
		errs = append(errs, e)
	}
	if len(errs) > 0 {
		return fmt.Sprintf(" error: %v", errs)
	}
	return ""
}

func (v Enum) String() string {
	switch v {
	case 0x0:
		return "0"
	case 0x1:
		return "1"
	case 0x2:
		return "2"
	case 0x3:
		return "LINE_STRIP"
	case 0x4:
		return "4"
	case 0x5:
		return "TRIANGLE_STRIP"
	case 0x6:
		return "TRIANGLE_FAN"
	case 0x300:
		return "SRC_COLOR"
	case 0x301:
		return "ONE_MINUS_SRC_COLOR"
	case 0x302:
		return "SRC_ALPHA"
	case 0x303:
		return "ONE_MINUS_SRC_ALPHA"
	case 0x304:
		return "DST_ALPHA"
	case 0x305:
		return "ONE_MINUS_DST_ALPHA"
	case 0x306:
		return "DST_COLOR"
	case 0x307:
		return "ONE_MINUS_DST_COLOR"
	case 0x308:
		return "SRC_ALPHA_SATURATE"
	case 0x8006:
		return "FUNC_ADD"
	case 0x8009:
		return "32777"
	case 0x883d:
		return "BLEND_EQUATION_ALPHA"
	case 0x800a:
		return "FUNC_SUBTRACT"
	case 0x800b:
		return "FUNC_REVERSE_SUBTRACT"
	case 0x80c8:
		return "BLEND_DST_RGB"
	case 0x80c9:
		return "BLEND_SRC_RGB"
	case 0x80ca:
		return "BLEND_DST_ALPHA"
	case 0x80cb:
		return "BLEND_SRC_ALPHA"
	case 0x8001:
		return "CONSTANT_COLOR"
	case 0x8002:
		return "ONE_MINUS_CONSTANT_COLOR"
	case 0x8003:
		return "CONSTANT_ALPHA"
	case 0x8004:
		return "ONE_MINUS_CONSTANT_ALPHA"
	case 0x8005:
		return "BLEND_COLOR"
	case 0x8892:
		return "ARRAY_BUFFER"
	case 0x8893:
		return "ELEMENT_ARRAY_BUFFER"
	case 0x8894:
		return "ARRAY_BUFFER_BINDING"
	case 0x8895:
		return "ELEMENT_ARRAY_BUFFER_BINDING"
	case 0x88e0:
		return "STREAM_DRAW"
	case 0x88e4:
		return "STATIC_DRAW"
	case 0x88e8:
		return "DYNAMIC_DRAW"
	case 0x8764:
		return "BUFFER_SIZE"
	case 0x8765:
		return "BUFFER_USAGE"
	case 0x8626:
		return "CURRENT_VERTEX_ATTRIB"
	case 0x404:
		return "FRONT"
	case 0x405:
		return "BACK"
	case 0x408:
		return "FRONT_AND_BACK"
	case 0xde1:
		return "TEXTURE_2D"
	case 0xb44:
		return "CULL_FACE"
	case 0xbe2:
		return "BLEND"
	case 0xbd0:
		return "DITHER"
	case 0xb90:
		return "STENCIL_TEST"
	case 0xb71:
		return "DEPTH_TEST"
	case 0xc11:
		return "SCISSOR_TEST"
	case 0x8037:
		return "POLYGON_OFFSET_FILL"
	case 0x809e:
		return "SAMPLE_ALPHA_TO_COVERAGE"
	case 0x80a0:
		return "SAMPLE_COVERAGE"
	case 0x500:
		return "INVALID_ENUM"
	case 0x501:
		return "INVALID_VALUE"
	case 0x502:
		return "INVALID_OPERATION"
	case 0x505:
		return "OUT_OF_MEMORY"
	case 0x900:
		return "CW"
	case 0x901:
		return "CCW"
	case 0xb21:
		return "LINE_WIDTH"
	case 0x846d:
		return "ALIASED_POINT_SIZE_RANGE"
	case 0x846e:
		return "ALIASED_LINE_WIDTH_RANGE"
	case 0xb45:
		return "CULL_FACE_MODE"
	case 0xb46:
		return "FRONT_FACE"
	case 0xb70:
		return "DEPTH_RANGE"
	case 0xb72:
		return "DEPTH_WRITEMASK"
	case 0xb73:
		return "DEPTH_CLEAR_VALUE"
	case 0xb74:
		return "DEPTH_FUNC"
	case 0xb91:
		return "STENCIL_CLEAR_VALUE"
	case 0xb92:
		return "STENCIL_FUNC"
	case 0xb94:
		return "STENCIL_FAIL"
	case 0xb95:
		return "STENCIL_PASS_DEPTH_FAIL"
	case 0xb96:
		return "STENCIL_PASS_DEPTH_PASS"
	case 0xb97:
		return "STENCIL_REF"
	case 0xb93:
		return "STENCIL_VALUE_MASK"
	case 0xb98:
		return "STENCIL_WRITEMASK"
	case 0x8800:
		return "STENCIL_BACK_FUNC"
	case 0x8801:
		return "STENCIL_BACK_FAIL"
	case 0x8802:
		return "STENCIL_BACK_PASS_DEPTH_FAIL"
	case 0x8803:
		return "STENCIL_BACK_PASS_DEPTH_PASS"
	case 0x8ca3:
		return "STENCIL_BACK_REF"
	case 0x8ca4:
		return "STENCIL_BACK_VALUE_MASK"
	case 0x8ca5:
		return "STENCIL_BACK_WRITEMASK"
	case 0xba2:
		return "VIEWPORT"
	case 0xc10:
		return "SCISSOR_BOX"
	case 0xc22:
		return "COLOR_CLEAR_VALUE"
	case 0xc23:
		return "COLOR_WRITEMASK"
	case 0xcf5:
		return "UNPACK_ALIGNMENT"
	case 0xd05:
		return "PACK_ALIGNMENT"
	case 0xd33:
		return "MAX_TEXTURE_SIZE"
	case 0xd3a:
		return "MAX_VIEWPORT_DIMS"
	case 0xd50:
		return "SUBPIXEL_BITS"
	case 0xd52:
		return "RED_BITS"
	case 0xd53:
		return "GREEN_BITS"
	case 0xd54:
		return "BLUE_BITS"
	case 0xd55:
		return "ALPHA_BITS"
	case 0xd56:
		return "DEPTH_BITS"
	case 0xd57:
		return "STENCIL_BITS"
	case 0x2a00:
		return "POLYGON_OFFSET_UNITS"
	case 0x8038:
		return "POLYGON_OFFSET_FACTOR"
	case 0x8069:
		return "TEXTURE_BINDING_2D"
	case 0x80a8:
		return "SAMPLE_BUFFERS"
	case 0x80a9:
		return "SAMPLES"
	case 0x80aa:
		return "SAMPLE_COVERAGE_VALUE"
	case 0x80ab:
		return "SAMPLE_COVERAGE_INVERT"
	case 0x86a2:
		return "NUM_COMPRESSED_TEXTURE_FORMATS"
	case 0x86a3:
		return "COMPRESSED_TEXTURE_FORMATS"
	case 0x1100:
		return "DONT_CARE"
	case 0x1101:
		return "FASTEST"
	case 0x1102:
		return "NICEST"
	case 0x8192:
		return "GENERATE_MIPMAP_HINT"
	case 0x1400:
		return "BYTE"
	case 0x1401:
		return "UNSIGNED_BYTE"
	case 0x1402:
		return "SHORT"
	case 0x1403:
		return "UNSIGNED_SHORT"
	case 0x1404:
		return "INT"
	case 0x1405:
		return "UNSIGNED_INT"
	case 0x1406:
		return "FLOAT"
	case 0x140c:
		return "FIXED"
	case 0x1902:
		return "DEPTH_COMPONENT"
	case 0x1906:
		return "ALPHA"
	case 0x1907:
		return "RGB"
	case 0x1908:
		return "RGBA"
	case 0x1909:
		return "LUMINANCE"
	case 0x190a:
		return "LUMINANCE_ALPHA"
	case 0x8033:
		return "UNSIGNED_SHORT_4_4_4_4"
	case 0x8034:
		return "UNSIGNED_SHORT_5_5_5_1"
	case 0x8363:
		return "UNSIGNED_SHORT_5_6_5"
	case 0x8869:
		return "MAX_VERTEX_ATTRIBS"
	case 0x8dfb:
		return "MAX_VERTEX_UNIFORM_VECTORS"
	case 0x8dfc:
		return "MAX_VARYING_VECTORS"
	case 0x8b4d:
		return "MAX_COMBINED_TEXTURE_IMAGE_UNITS"
	case 0x8b4c:
		return "MAX_VERTEX_TEXTURE_IMAGE_UNITS"
	case 0x8872:
		return "MAX_TEXTURE_IMAGE_UNITS"
	case 0x8dfd:
		return "MAX_FRAGMENT_UNIFORM_VECTORS"
	case 0x8b4f:
		return "SHADER_TYPE"
	case 0x8b80:
		return "DELETE_STATUS"
	case 0x8b82:
		return "LINK_STATUS"
	case 0x8b83:
		return "VALIDATE_STATUS"
	case 0x8b85:
		return "ATTACHED_SHADERS"
	case 0x8b86:
		return "ACTIVE_UNIFORMS"
	case 0x8b87:
		return "ACTIVE_UNIFORM_MAX_LENGTH"
	case 0x8b89:
		return "ACTIVE_ATTRIBUTES"
	case 0x8b8a:
		return "ACTIVE_ATTRIBUTE_MAX_LENGTH"
	case 0x8b8c:
		return "SHADING_LANGUAGE_VERSION"
	case 0x8b8d:
		return "CURRENT_PROGRAM"
	case 0x200:
		return "NEVER"
	case 0x201:
		return "LESS"
	case 0x202:
		return "EQUAL"
	case 0x203:
		return "LEQUAL"
	case 0x204:
		return "GREATER"
	case 0x205:
		return "NOTEQUAL"
	case 0x206:
		return "GEQUAL"
	case 0x207:
		return "ALWAYS"
	case 0x1e00:
		return "KEEP"
	case 0x1e01:
		return "REPLACE"
	case 0x1e02:
		return "INCR"
	case 0x1e03:
		return "DECR"
	case 0x150a:
		return "INVERT"
	case 0x8507:
		return "INCR_WRAP"
	case 0x8508:
		return "DECR_WRAP"
	case 0x1f00:
		return "VENDOR"
	case 0x1f01:
		return "RENDERER"
	case 0x1f02:
		return "VERSION"
	case 0x1f03:
		return "EXTENSIONS"
	case 0x2600:
		return "NEAREST"
	case 0x2601:
		return "LINEAR"
	case 0x2700:
		return "NEAREST_MIPMAP_NEAREST"
	case 0x2701:
		return "LINEAR_MIPMAP_NEAREST"
	case 0x2702:
		return "NEAREST_MIPMAP_LINEAR"
	case 0x2703:
		return "LINEAR_MIPMAP_LINEAR"
	case 0x2800:
		return "TEXTURE_MAG_FILTER"
	case 0x2801:
		return "TEXTURE_MIN_FILTER"
	case 0x2802:
		return "TEXTURE_WRAP_S"
	case 0x2803:
		return "TEXTURE_WRAP_T"
	case 0x1702:
		return "TEXTURE"
	case 0x8513:
		return "TEXTURE_CUBE_MAP"
	case 0x8514:
		return "TEXTURE_BINDING_CUBE_MAP"
	case 0x8515:
		return "TEXTURE_CUBE_MAP_POSITIVE_X"
	case 0x8516:
		return "TEXTURE_CUBE_MAP_NEGATIVE_X"
	case 0x8517:
		return "TEXTURE_CUBE_MAP_POSITIVE_Y"
	case 0x8518:
		return "TEXTURE_CUBE_MAP_NEGATIVE_Y"
	case 0x8519:
		return "TEXTURE_CUBE_MAP_POSITIVE_Z"
	case 0x851a:
		return "TEXTURE_CUBE_MAP_NEGATIVE_Z"
	case 0x851c:
		return "MAX_CUBE_MAP_TEXTURE_SIZE"
	case 0x84c0:
		return "TEXTURE0"
	case 0x84c1:
		return "TEXTURE1"
	case 0x84c2:
		return "TEXTURE2"
	case 0x84c3:
		return "TEXTURE3"
	case 0x84c4:
		return "TEXTURE4"
	case 0x84c5:
		return "TEXTURE5"
	case 0x84c6:
		return "TEXTURE6"
	case 0x84c7:
		return "TEXTURE7"
	case 0x84c8:
		return "TEXTURE8"
	case 0x84c9:
		return "TEXTURE9"
	case 0x84ca:
		return "TEXTURE10"
	case 0x84cb:
		return "TEXTURE11"
	case 0x84cc:
		return "TEXTURE12"
	case 0x84cd:
		return "TEXTURE13"
	case 0x84ce:
		return "TEXTURE14"
	case 0x84cf:
		return "TEXTURE15"
	case 0x84d0:
		return "TEXTURE16"
	case 0x84d1:
		return "TEXTURE17"
	case 0x84d2:
		return "TEXTURE18"
	case 0x84d3:
		return "TEXTURE19"
	case 0x84d4:
		return "TEXTURE20"
	case 0x84d5:
		return "TEXTURE21"
	case 0x84d6:
		return "TEXTURE22"
	case 0x84d7:
		return "TEXTURE23"
	case 0x84d8:
		return "TEXTURE24"
	case 0x84d9:
		return "TEXTURE25"
	case 0x84da:
		return "TEXTURE26"
	case 0x84db:
		return "TEXTURE27"
	case 0x84dc:
		return "TEXTURE28"
	case 0x84dd:
		return "TEXTURE29"
	case 0x84de:
		return "TEXTURE30"
	case 0x84df:
		return "TEXTURE31"
	case 0x84e0:
		return "ACTIVE_TEXTURE"
	case 0x2901:
		return "REPEAT"
	case 0x812f:
		return "CLAMP_TO_EDGE"
	case 0x8370:
		return "MIRRORED_REPEAT"
	case 0x8622:
		return "VERTEX_ATTRIB_ARRAY_ENABLED"
	case 0x8623:
		return "VERTEX_ATTRIB_ARRAY_SIZE"
	case 0x8624:
		return "VERTEX_ATTRIB_ARRAY_STRIDE"
	case 0x8625:
		return "VERTEX_ATTRIB_ARRAY_TYPE"
	case 0x886a:
		return "VERTEX_ATTRIB_ARRAY_NORMALIZED"
	case 0x8645:
		return "VERTEX_ATTRIB_ARRAY_POINTER"
	case 0x889f:
		return "VERTEX_ATTRIB_ARRAY_BUFFER_BINDING"
	case 0x8b9a:
		return "IMPLEMENTATION_COLOR_READ_TYPE"
	case 0x8b9b:
		return "IMPLEMENTATION_COLOR_READ_FORMAT"
	case 0x8b81:
		return "COMPILE_STATUS"
	case 0x8b84:
		return "INFO_LOG_LENGTH"
	case 0x8b88:
		return "SHADER_SOURCE_LENGTH"
	case 0x8dfa:
		return "SHADER_COMPILER"
	case 0x8df8:
		return "SHADER_BINARY_FORMATS"
	case 0x8df9:
		return "NUM_SHADER_BINARY_FORMATS"
	case 0x8df0:
		return "LOW_FLOAT"
	case 0x8df1:
		return "MEDIUM_FLOAT"
	case 0x8df2:
		return "HIGH_FLOAT"
	case 0x8df3:
		return "LOW_INT"
	case 0x8df4:
		return "MEDIUM_INT"
	case 0x8df5:
		return "HIGH_INT"
	case 0x8d40:
		return "FRAMEBUFFER"
	case 0x8d41:
		return "RENDERBUFFER"
	case 0x8056:
		return "RGBA4"
	case 0x8057:
		return "RGB5_A1"
	case 0x8d62:
		return "RGB565"
	case 0x81a5:
		return "DEPTH_COMPONENT16"
	case 0x8d48:
		return "STENCIL_INDEX8"
	case 0x8d42:
		return "RENDERBUFFER_WIDTH"
	case 0x8d43:
		return "RENDERBUFFER_HEIGHT"
	case 0x8d44:
		return "RENDERBUFFER_INTERNAL_FORMAT"
	case 0x8d50:
		return "RENDERBUFFER_RED_SIZE"
	case 0x8d51:
		return "RENDERBUFFER_GREEN_SIZE"
	case 0x8d52:
		return "RENDERBUFFER_BLUE_SIZE"
	case 0x8d53:
		return "RENDERBUFFER_ALPHA_SIZE"
	case 0x8d54:
		return "RENDERBUFFER_DEPTH_SIZE"
	case 0x8d55:
		return "RENDERBUFFER_STENCIL_SIZE"
	case 0x8cd0:
		return "FRAMEBUFFER_ATTACHMENT_OBJECT_TYPE"
	case 0x8cd1:
		return "FRAMEBUFFER_ATTACHMENT_OBJECT_NAME"
	case 0x8cd2:
		return "FRAMEBUFFER_ATTACHMENT_TEXTURE_LEVEL"
	case 0x8cd3:
		return "FRAMEBUFFER_ATTACHMENT_TEXTURE_CUBE_MAP_FACE"
	case 0x8ce0:
		return "COLOR_ATTACHMENT0"
	case 0x8d00:
		return "DEPTH_ATTACHMENT"
	case 0x8d20:
		return "STENCIL_ATTACHMENT"
	case 0x8cd5:
		return "FRAMEBUFFER_COMPLETE"
	case 0x8cd6:
		return "FRAMEBUFFER_INCOMPLETE_ATTACHMENT"
	case 0x8cd7:
		return "FRAMEBUFFER_INCOMPLETE_MISSING_ATTACHMENT"
	case 0x8cd9:
		return "FRAMEBUFFER_INCOMPLETE_DIMENSIONS"
	case 0x8cdd:
		return "FRAMEBUFFER_UNSUPPORTED"
	case 0x8ca6:
		return "36006"
	case 0x8ca7:
		return "RENDERBUFFER_BINDING"
	case 0x84e8:
		return "MAX_RENDERBUFFER_SIZE"
	case 0x506:
		return "INVALID_FRAMEBUFFER_OPERATION"
	case 0x100:
		return "DEPTH_BUFFER_BIT"
	case 0x400:
		return "STENCIL_BUFFER_BIT"
	case 0x4000:
		return "COLOR_BUFFER_BIT"
	case 0x8b50:
		return "FLOAT_VEC2"
	case 0x8b51:
		return "FLOAT_VEC3"
	case 0x8b52:
		return "FLOAT_VEC4"
	case 0x8b53:
		return "INT_VEC2"
	case 0x8b54:
		return "INT_VEC3"
	case 0x8b55:
		return "INT_VEC4"
	case 0x8b56:
		return "BOOL"
	case 0x8b57:
		return "BOOL_VEC2"
	case 0x8b58:
		return "BOOL_VEC3"
	case 0x8b59:
		return "BOOL_VEC4"
	case 0x8b5a:
		return "FLOAT_MAT2"
	case 0x8b5b:
		return "FLOAT_MAT3"
	case 0x8b5c:
		return "FLOAT_MAT4"
	case 0x8b5e:
		return "SAMPLER_2D"
	case 0x8b60:
		return "SAMPLER_CUBE"
	case 0x8b30:
		return "FRAGMENT_SHADER"
	case 0x8b31:
		return "VERTEX_SHADER"
	case 0x8a35:
		return "ACTIVE_UNIFORM_BLOCK_MAX_NAME_LENGTH"
	case 0x8a36:
		return "ACTIVE_UNIFORM_BLOCKS"
	case 0x911a:
		return "ALREADY_SIGNALED"
	case 0x8c2f:
		return "ANY_SAMPLES_PASSED"
	case 0x8d6a:
		return "ANY_SAMPLES_PASSED_CONSERVATIVE"
	case 0x1905:
		return "BLUE"
	case 0x911f:
		return "BUFFER_ACCESS_FLAGS"
	case 0x9120:
		return "BUFFER_MAP_LENGTH"
	case 0x9121:
		return "BUFFER_MAP_OFFSET"
	case 0x88bc:
		return "BUFFER_MAPPED"
	case 0x88bd:
		return "BUFFER_MAP_POINTER"
	case 0x1800:
		return "COLOR"
	case 0x8cea:
		return "COLOR_ATTACHMENT10"
	case 0x8ce1:
		return "COLOR_ATTACHMENT1"
	case 0x8ceb:
		return "COLOR_ATTACHMENT11"
	case 0x8cec:
		return "COLOR_ATTACHMENT12"
	case 0x8ced:
		return "COLOR_ATTACHMENT13"
	case 0x8cee:
		return "COLOR_ATTACHMENT14"
	case 0x8cef:
		return "COLOR_ATTACHMENT15"
	case 0x8ce2:
		return "COLOR_ATTACHMENT2"
	case 0x8ce3:
		return "COLOR_ATTACHMENT3"
	case 0x8ce4:
		return "COLOR_ATTACHMENT4"
	case 0x8ce5:
		return "COLOR_ATTACHMENT5"
	case 0x8ce6:
		return "COLOR_ATTACHMENT6"
	case 0x8ce7:
		return "COLOR_ATTACHMENT7"
	case 0x8ce8:
		return "COLOR_ATTACHMENT8"
	case 0x8ce9:
		return "COLOR_ATTACHMENT9"
	case 0x884e:
		return "COMPARE_REF_TO_TEXTURE"
	case 0x9270:
		return "COMPRESSED_R11_EAC"
	case 0x9272:
		return "COMPRESSED_RG11_EAC"
	case 0x9274:
		return "COMPRESSED_RGB8_ETC2"
	case 0x9276:
		return "COMPRESSED_RGB8_PUNCHTHROUGH_ALPHA1_ETC2"
	case 0x9278:
		return "COMPRESSED_RGBA8_ETC2_EAC"
	case 0x9271:
		return "COMPRESSED_SIGNED_R11_EAC"
	case 0x9273:
		return "COMPRESSED_SIGNED_RG11_EAC"
	case 0x9279:
		return "COMPRESSED_SRGB8_ALPHA8_ETC2_EAC"
	case 0x9275:
		return "COMPRESSED_SRGB8_ETC2"
	case 0x9277:
		return "COMPRESSED_SRGB8_PUNCHTHROUGH_ALPHA1_ETC2"
	case 0x911c:
		return "CONDITION_SATISFIED"
	case 0x8f36:
		return "36662"
	case 0x8f37:
		return "36663"
	case 0x8865:
		return "CURRENT_QUERY"
	case 0x1801:
		return "DEPTH"
	case 0x88f0:
		return "DEPTH24_STENCIL8"
	case 0x8cad:
		return "DEPTH32F_STENCIL8"
	case 0x81a6:
		return "DEPTH_COMPONENT24"
	case 0x8cac:
		return "DEPTH_COMPONENT32F"
	case 0x84f9:
		return "DEPTH_STENCIL"
	case 0x821a:
		return "DEPTH_STENCIL_ATTACHMENT"
	case 0x8825:
		return "DRAW_BUFFER0"
	case 0x882f:
		return "DRAW_BUFFER10"
	case 0x8826:
		return "DRAW_BUFFER1"
	case 0x8830:
		return "DRAW_BUFFER11"
	case 0x8831:
		return "DRAW_BUFFER12"
	case 0x8832:
		return "DRAW_BUFFER13"
	case 0x8833:
		return "DRAW_BUFFER14"
	case 0x8834:
		return "DRAW_BUFFER15"
	case 0x8827:
		return "DRAW_BUFFER2"
	case 0x8828:
		return "DRAW_BUFFER3"
	case 0x8829:
		return "DRAW_BUFFER4"
	case 0x882a:
		return "DRAW_BUFFER5"
	case 0x882b:
		return "DRAW_BUFFER6"
	case 0x882c:
		return "DRAW_BUFFER7"
	case 0x882d:
		return "DRAW_BUFFER8"
	case 0x882e:
		return "DRAW_BUFFER9"
	case 0x8ca9:
		return "DRAW_FRAMEBUFFER"
	case 0x88ea:
		return "DYNAMIC_COPY"
	case 0x88e9:
		return "DYNAMIC_READ"
	case 0x8dad:
		return "FLOAT_32_UNSIGNED_INT_24_8_REV"
	case 0x8b65:
		return "FLOAT_MAT2x3"
	case 0x8b66:
		return "FLOAT_MAT2x4"
	case 0x8b67:
		return "FLOAT_MAT3x2"
	case 0x8b68:
		return "FLOAT_MAT3x4"
	case 0x8b69:
		return "FLOAT_MAT4x2"
	case 0x8b6a:
		return "FLOAT_MAT4x3"
	case 0x8b8b:
		return "FRAGMENT_SHADER_DERIVATIVE_HINT"
	case 0x8215:
		return "FRAMEBUFFER_ATTACHMENT_ALPHA_SIZE"
	case 0x8214:
		return "FRAMEBUFFER_ATTACHMENT_BLUE_SIZE"
	case 0x8210:
		return "FRAMEBUFFER_ATTACHMENT_COLOR_ENCODING"
	case 0x8211:
		return "FRAMEBUFFER_ATTACHMENT_COMPONENT_TYPE"
	case 0x8216:
		return "FRAMEBUFFER_ATTACHMENT_DEPTH_SIZE"
	case 0x8213:
		return "FRAMEBUFFER_ATTACHMENT_GREEN_SIZE"
	case 0x8212:
		return "FRAMEBUFFER_ATTACHMENT_RED_SIZE"
	case 0x8217:
		return "FRAMEBUFFER_ATTACHMENT_STENCIL_SIZE"
	case 0x8cd4:
		return "FRAMEBUFFER_ATTACHMENT_TEXTURE_LAYER"
	case 0x8218:
		return "FRAMEBUFFER_DEFAULT"
	case 0x8d56:
		return "FRAMEBUFFER_INCOMPLETE_MULTISAMPLE"
	case 0x8219:
		return "FRAMEBUFFER_UNDEFINED"
	case 0x1904:
		return "GREEN"
	case 0x140b:
		return "HALF_FLOAT"
	case 0x8d9f:
		return "INT_2_10_10_10_REV"
	case 0x8c8c:
		return "INTERLEAVED_ATTRIBS"
	case 0x8dca:
		return "INT_SAMPLER_2D"
	case 0x8dcf:
		return "INT_SAMPLER_2D_ARRAY"
	case 0x8dcb:
		return "INT_SAMPLER_3D"
	case 0x8dcc:
		return "INT_SAMPLER_CUBE"
	case 0xffffffff:
		return "INVALID_INDEX"
	case 0x821b:
		return "MAJOR_VERSION"
	case 0x10:
		return "MAP_FLUSH_EXPLICIT_BIT"
	case 0x8:
		return "MAP_INVALIDATE_BUFFER_BIT"
	case 0x20:
		return "MAP_UNSYNCHRONIZED_BIT"
	case 0x8008:
		return "MAX"
	case 0x8073:
		return "MAX_3D_TEXTURE_SIZE"
	case 0x88ff:
		return "MAX_ARRAY_TEXTURE_LAYERS"
	case 0x8cdf:
		return "MAX_COLOR_ATTACHMENTS"
	case 0x8a33:
		return "MAX_COMBINED_FRAGMENT_UNIFORM_COMPONENTS"
	case 0x8a2e:
		return "MAX_COMBINED_UNIFORM_BLOCKS"
	case 0x8a31:
		return "MAX_COMBINED_VERTEX_UNIFORM_COMPONENTS"
	case 0x8824:
		return "MAX_DRAW_BUFFERS"
	case 0x8d6b:
		return "MAX_ELEMENT_INDEX"
	case 0x80e9:
		return "MAX_ELEMENTS_INDICES"
	case 0x80e8:
		return "MAX_ELEMENTS_VERTICES"
	case 0x9125:
		return "MAX_FRAGMENT_INPUT_COMPONENTS"
	case 0x8a2d:
		return "MAX_FRAGMENT_UNIFORM_BLOCKS"
	case 0x8b49:
		return "MAX_FRAGMENT_UNIFORM_COMPONENTS"
	case 0x8905:
		return "MAX_PROGRAM_TEXEL_OFFSET"
	case 0x8d57:
		return "MAX_SAMPLES"
	case 0x9111:
		return "MAX_SERVER_WAIT_TIMEOUT"
	case 0x84fd:
		return "MAX_TEXTURE_LOD_BIAS"
	case 0x8c8a:
		return "MAX_TRANSFORM_FEEDBACK_INTERLEAVED_COMPONENTS"
	case 0x8c8b:
		return "MAX_TRANSFORM_FEEDBACK_SEPARATE_ATTRIBS"
	case 0x8c80:
		return "MAX_TRANSFORM_FEEDBACK_SEPARATE_COMPONENTS"
	case 0x8a30:
		return "MAX_UNIFORM_BLOCK_SIZE"
	case 0x8a2f:
		return "MAX_UNIFORM_BUFFER_BINDINGS"
	case 0x8b4b:
		return "MAX_VARYING_COMPONENTS"
	case 0x9122:
		return "MAX_VERTEX_OUTPUT_COMPONENTS"
	case 0x8a2b:
		return "MAX_VERTEX_UNIFORM_BLOCKS"
	case 0x8b4a:
		return "MAX_VERTEX_UNIFORM_COMPONENTS"
	case 0x8007:
		return "MIN"
	case 0x821c:
		return "MINOR_VERSION"
	case 0x8904:
		return "MIN_PROGRAM_TEXEL_OFFSET"
	case 0x821d:
		return "NUM_EXTENSIONS"
	case 0x87fe:
		return "NUM_PROGRAM_BINARY_FORMATS"
	case 0x9380:
		return "NUM_SAMPLE_COUNTS"
	case 0x9112:
		return "OBJECT_TYPE"
	case 0xd02:
		return "PACK_ROW_LENGTH"
	case 0xd04:
		return "PACK_SKIP_PIXELS"
	case 0xd03:
		return "PACK_SKIP_ROWS"
	case 0x88eb:
		return "PIXEL_PACK_BUFFER"
	case 0x88ed:
		return "PIXEL_PACK_BUFFER_BINDING"
	case 0x88ec:
		return "PIXEL_UNPACK_BUFFER"
	case 0x88ef:
		return "PIXEL_UNPACK_BUFFER_BINDING"
	case 0x8d69:
		return "PRIMITIVE_RESTART_FIXED_INDEX"
	case 0x87ff:
		return "PROGRAM_BINARY_FORMATS"
	case 0x8741:
		return "PROGRAM_BINARY_LENGTH"
	case 0x8257:
		return "PROGRAM_BINARY_RETRIEVABLE_HINT"
	case 0x8866:
		return "QUERY_RESULT"
	case 0x8867:
		return "QUERY_RESULT_AVAILABLE"
	case 0x8c3a:
		return "R11F_G11F_B10F"
	case 0x822d:
		return "R16F"
	case 0x8233:
		return "R16I"
	case 0x8234:
		return "R16UI"
	case 0x822e:
		return "R32F"
	case 0x8235:
		return "R32I"
	case 0x8236:
		return "R32UI"
	case 0x8229:
		return "R8"
	case 0x8231:
		return "R8I"
	case 0x8f94:
		return "R8_SNORM"
	case 0x8232:
		return "R8UI"
	case 0x8c89:
		return "RASTERIZER_DISCARD"
	case 0xc02:
		return "READ_BUFFER"
	case 0x8ca8:
		return "READ_FRAMEBUFFER"
	case 0x8caa:
		return "READ_FRAMEBUFFER_BINDING"
	case 0x1903:
		return "RED"
	case 0x8d94:
		return "RED_INTEGER"
	case 0x8cab:
		return "RENDERBUFFER_SAMPLES"
	case 0x8227:
		return "RG"
	case 0x822f:
		return "RG16F"
	case 0x8239:
		return "RG16I"
	case 0x823a:
		return "RG16UI"
	case 0x8230:
		return "RG32F"
	case 0x823b:
		return "RG32I"
	case 0x823c:
		return "RG32UI"
	case 0x822b:
		return "RG8"
	case 0x8237:
		return "RG8I"
	case 0x8f95:
		return "RG8_SNORM"
	case 0x8238:
		return "RG8UI"
	case 0x8059:
		return "RGB10_A2"
	case 0x906f:
		return "RGB10_A2UI"
	case 0x881b:
		return "RGB16F"
	case 0x8d89:
		return "RGB16I"
	case 0x8d77:
		return "RGB16UI"
	case 0x8815:
		return "RGB32F"
	case 0x8d83:
		return "RGB32I"
	case 0x8d71:
		return "RGB32UI"
	case 0x8051:
		return "RGB8"
	case 0x8d8f:
		return "RGB8I"
	case 0x8f96:
		return "RGB8_SNORM"
	case 0x8d7d:
		return "RGB8UI"
	case 0x8c3d:
		return "RGB9_E5"
	case 0x881a:
		return "RGBA16F"
	case 0x8d88:
		return "RGBA16I"
	case 0x8d76:
		return "RGBA16UI"
	case 0x8814:
		return "RGBA32F"
	case 0x8d82:
		return "RGBA32I"
	case 0x8d70:
		return "RGBA32UI"
	case 0x8058:
		return "RGBA8"
	case 0x8d8e:
		return "RGBA8I"
	case 0x8f97:
		return "RGBA8_SNORM"
	case 0x8d7c:
		return "RGBA8UI"
	case 0x8d99:
		return "RGBA_INTEGER"
	case 0x8d98:
		return "RGB_INTEGER"
	case 0x8228:
		return "RG_INTEGER"
	case 0x8dc1:
		return "SAMPLER_2D_ARRAY"
	case 0x8dc4:
		return "SAMPLER_2D_ARRAY_SHADOW"
	case 0x8b62:
		return "SAMPLER_2D_SHADOW"
	case 0x8b5f:
		return "SAMPLER_3D"
	case 0x8919:
		return "SAMPLER_BINDING"
	case 0x8dc5:
		return "SAMPLER_CUBE_SHADOW"
	case 0x8c8d:
		return "SEPARATE_ATTRIBS"
	case 0x9119:
		return "SIGNALED"
	case 0x8f9c:
		return "SIGNED_NORMALIZED"
	case 0x8c40:
		return "SRGB"
	case 0x8c41:
		return "SRGB8"
	case 0x8c43:
		return "SRGB8_ALPHA8"
	case 0x88e6:
		return "STATIC_COPY"
	case 0x88e5:
		return "STATIC_READ"
	case 0x1802:
		return "STENCIL"
	case 0x88e2:
		return "STREAM_COPY"
	case 0x88e1:
		return "STREAM_READ"
	case 0x9113:
		return "SYNC_CONDITION"
	case 0x9116:
		return "SYNC_FENCE"
	case 0x9115:
		return "SYNC_FLAGS"
	case 0x9117:
		return "SYNC_GPU_COMMANDS_COMPLETE"
	case 0x9114:
		return "SYNC_STATUS"
	case 0x8c1a:
		return "TEXTURE_2D_ARRAY"
	case 0x806f:
		return "TEXTURE_3D"
	case 0x813c:
		return "TEXTURE_BASE_LEVEL"
	case 0x8c1d:
		return "TEXTURE_BINDING_2D_ARRAY"
	case 0x806a:
		return "TEXTURE_BINDING_3D"
	case 0x884d:
		return "TEXTURE_COMPARE_FUNC"
	case 0x884c:
		return "TEXTURE_COMPARE_MODE"
	case 0x912f:
		return "TEXTURE_IMMUTABLE_FORMAT"
	case 0x82df:
		return "TEXTURE_IMMUTABLE_LEVELS"
	case 0x813d:
		return "TEXTURE_MAX_LEVEL"
	case 0x813b:
		return "TEXTURE_MAX_LOD"
	case 0x813a:
		return "TEXTURE_MIN_LOD"
	case 0x8e45:
		return "TEXTURE_SWIZZLE_A"
	case 0x8e44:
		return "TEXTURE_SWIZZLE_B"
	case 0x8e43:
		return "TEXTURE_SWIZZLE_G"
	case 0x8e42:
		return "TEXTURE_SWIZZLE_R"
	case 0x8072:
		return "TEXTURE_WRAP_R"
	case 0x911b:
		return "TIMEOUT_EXPIRED"
	case 0x8e22:
		return "TRANSFORM_FEEDBACK"
	case 0x8e24:
		return "TRANSFORM_FEEDBACK_ACTIVE"
	case 0x8e25:
		return "TRANSFORM_FEEDBACK_BINDING"
	case 0x8c8e:
		return "TRANSFORM_FEEDBACK_BUFFER"
	case 0x8c8f:
		return "TRANSFORM_FEEDBACK_BUFFER_BINDING"
	case 0x8c7f:
		return "TRANSFORM_FEEDBACK_BUFFER_MODE"
	case 0x8c85:
		return "TRANSFORM_FEEDBACK_BUFFER_SIZE"
	case 0x8c84:
		return "TRANSFORM_FEEDBACK_BUFFER_START"
	case 0x8e23:
		return "TRANSFORM_FEEDBACK_PAUSED"
	case 0x8c88:
		return "TRANSFORM_FEEDBACK_PRIMITIVES_WRITTEN"
	case 0x8c76:
		return "TRANSFORM_FEEDBACK_VARYING_MAX_LENGTH"
	case 0x8c83:
		return "TRANSFORM_FEEDBACK_VARYINGS"
	case 0x8a3c:
		return "UNIFORM_ARRAY_STRIDE"
	case 0x8a43:
		return "UNIFORM_BLOCK_ACTIVE_UNIFORM_INDICES"
	case 0x8a42:
		return "UNIFORM_BLOCK_ACTIVE_UNIFORMS"
	case 0x8a3f:
		return "UNIFORM_BLOCK_BINDING"
	case 0x8a40:
		return "UNIFORM_BLOCK_DATA_SIZE"
	case 0x8a3a:
		return "UNIFORM_BLOCK_INDEX"
	case 0x8a41:
		return "UNIFORM_BLOCK_NAME_LENGTH"
	case 0x8a46:
		return "UNIFORM_BLOCK_REFERENCED_BY_FRAGMENT_SHADER"
	case 0x8a44:
		return "UNIFORM_BLOCK_REFERENCED_BY_VERTEX_SHADER"
	case 0x8a11:
		return "UNIFORM_BUFFER"
	case 0x8a28:
		return "UNIFORM_BUFFER_BINDING"
	case 0x8a34:
		return "UNIFORM_BUFFER_OFFSET_ALIGNMENT"
	case 0x8a2a:
		return "UNIFORM_BUFFER_SIZE"
	case 0x8a29:
		return "UNIFORM_BUFFER_START"
	case 0x8a3e:
		return "UNIFORM_IS_ROW_MAJOR"
	case 0x8a3d:
		return "UNIFORM_MATRIX_STRIDE"
	case 0x8a39:
		return "UNIFORM_NAME_LENGTH"
	case 0x8a3b:
		return "UNIFORM_OFFSET"
	case 0x8a38:
		return "UNIFORM_SIZE"
	case 0x8a37:
		return "UNIFORM_TYPE"
	case 0x806e:
		return "UNPACK_IMAGE_HEIGHT"
	case 0xcf2:
		return "UNPACK_ROW_LENGTH"
	case 0x806d:
		return "UNPACK_SKIP_IMAGES"
	case 0xcf4:
		return "UNPACK_SKIP_PIXELS"
	case 0xcf3:
		return "UNPACK_SKIP_ROWS"
	case 0x9118:
		return "UNSIGNALED"
	case 0x8c3b:
		return "UNSIGNED_INT_10F_11F_11F_REV"
	case 0x8368:
		return "UNSIGNED_INT_2_10_10_10_REV"
	case 0x84fa:
		return "UNSIGNED_INT_24_8"
	case 0x8c3e:
		return "UNSIGNED_INT_5_9_9_9_REV"
	case 0x8dd2:
		return "UNSIGNED_INT_SAMPLER_2D"
	case 0x8dd7:
		return "UNSIGNED_INT_SAMPLER_2D_ARRAY"
	case 0x8dd3:
		return "UNSIGNED_INT_SAMPLER_3D"
	case 0x8dd4:
		return "UNSIGNED_INT_SAMPLER_CUBE"
	case 0x8dc6:
		return "UNSIGNED_INT_VEC2"
	case 0x8dc7:
		return "UNSIGNED_INT_VEC3"
	case 0x8dc8:
		return "UNSIGNED_INT_VEC4"
	case 0x8c17:
		return "UNSIGNED_NORMALIZED"
	case 0x85b5:
		return "VERTEX_ARRAY_BINDING"
	case 0x88fe:
		return "VERTEX_ATTRIB_ARRAY_DIVISOR"
	case 0x88fd:
		return "VERTEX_ATTRIB_ARRAY_INTEGER"
	case 0x911d:
		return "WAIT_FAILED"
	default:
		return fmt.Sprintf("gl.Enum(0x%x)", uint32(v))
	}
}

func (ctx *context) ActiveTexture(texture Enum) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.ActiveTexture(%v) %v", texture, errstr)
	}()
	ctx.Call("activateTexture", texture.c())
}

func (ctx *context) AttachShader(p Program, s Shader) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.AttachShader(%v, %v) %v", p, s, errstr)
	}()
	ctx.Call("attachShader", p.c(), s.c())
}

func (ctx *context) BindAttribLocation(p Program, a Attrib, name string) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.BindAttribLocation(%v, %v, %v) %v", p, a, name, errstr)
	}()
	ctx.Call("bindAttribLocation", p.c(), a.c(), name)
}

func (ctx *context) BindBuffer(target Enum, b Buffer) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.BindBuffer(%v, %v) %v", target, b, errstr)
	}()
	ctx.Call("bindBuffer", target.c(), b.c())
}

func (ctx *context) BindFramebuffer(target Enum, fb Framebuffer) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.BindFramebuffer(%v, %v) %v", target, fb, errstr)
	}()
	ctx.Call("bindFramebuffer", target.c(), fb.c())
}

func (ctx *context) BindRenderbuffer(target Enum, rb Renderbuffer) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.BindRenderbuffer(%v, %v) %v", target, rb, errstr)
	}()
	ctx.Call("bindRenderbuffer", target.c(), rb.c())
}

func (ctx *context) BindTexture(target Enum, t Texture) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.BindTexture(%v, %v) %v", target, t, errstr)
	}()
	ctx.Call("bindTexture", target.c(), t.c())
}

func (ctx *context) BindVertexArray(va VertexArray) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.BindVertexArray(%v) %v", va, errstr)
	}()
	ctx.callWebGL2Compat("OES_vertex_array_object", "OES", "bindVertexArray", va.c())
}

func (ctx *context) BlendColor(red, green, blue, alpha float32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.BlendColor(%v, %v, %v, %v) %v", red, green, blue, alpha, errstr)
	}()
	ctx.Call("blendColor", red, green, blue, alpha)
}

func (ctx *context) BlendEquation(mode Enum) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.BlendEquation(%v) %v", mode, errstr)
	}()
	ctx.Call("blendEquation", mode.c())
}

func (ctx *context) BlendEquationSeparate(modeRGB, modeAlpha Enum) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.BlendEquationSeparate(%v, %v) %v", modeRGB, modeAlpha, errstr)
	}()
	ctx.Call("blendEquationSeparate", modeRGB.c(), modeAlpha.c())
}

func (ctx *context) BlendFunc(sfactor, dfactor Enum) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.BlendFunc(%v, %v) %v", sfactor, dfactor, errstr)
	}()
	ctx.Call("blendFunc", sfactor.c(), dfactor.c())
}

func (ctx *context) BlendFuncSeparate(sfactorRGB, dfactorRGB, sfactorAlpha, dfactorAlpha Enum) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.BlendFuncSeparate(%v, %v, %v, %v) %v", sfactorRGB, dfactorRGB, sfactorAlpha, dfactorAlpha, errstr)
	}()
	ctx.Call("blendFuncSeparate", sfactorRGB.c(), dfactorRGB.c(), sfactorAlpha.c(), dfactorAlpha.c())
}

func (ctx *context) BufferData(target Enum, src []byte, usage Enum) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.BufferData(%v, len(%d), %v) %v", target, len(src), usage, errstr)
	}()
	ctx.Call("bindData", target.c(), js.NewArrayBuffer(src), usage.c())
}

func (ctx *context) BufferInit(target Enum, size int, usage Enum) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.BufferInit(%v, %v, %v) %v", target, size, usage, errstr)
	}()
	ctx.Call("bufferData", target.c(), size, usage.c())
}

func (ctx *context) BufferSubData(target Enum, offset int, data []byte) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.BufferSubData(%v, %v, len(%d)) %v", target, offset, len(data), errstr)
	}()
	ctx.Call("bufferSubData", target.c(), offset, js.NewArrayBuffer(data))
}

func (ctx *context) CheckFramebufferStatus(target Enum) (r0 Enum) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.CheckFramebufferStatus(%v) %v%v", target, r0, errstr)
	}()
	return Enum(ctx.Call("checkFramebufferStatus", target.c()).Int())
}

func (ctx *context) Clear(mask Enum) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.Clear(%v) %v", mask, errstr)
	}()
	ctx.Call("clear", mask)
}

func (ctx *context) ClearColor(red, green, blue, alpha float32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.ClearColor(%v, %v, %v, %v) %v", red, green, blue, alpha, errstr)
	}()
	ctx.Call("clearColor", red, green, blue, alpha)
}

func (ctx *context) ClearDepthf(d float32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.ClearDepthf(%v) %v", d, errstr)
	}()
	ctx.Call("clearDepth", d)
}

func (ctx *context) ClearStencil(s int) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.ClearStencil(%v) %v", s, errstr)
	}()
	ctx.Call("clearStencil", s)
}

func (ctx *context) ColorMask(red, green, blue, alpha bool) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.ColorMask(%v, %v, %v, %v) %v", red, green, blue, alpha, errstr)
	}()
	ctx.Call("colorMask", red, green, blue, alpha)
}

func (ctx *context) CompileShader(s Shader) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.CompileShader(%v) %v", s, errstr)
	}()
	ctx.Call("compileShader", s.c())
}

func (ctx *context) CompressedTexImage2D(target Enum, level int, internalformat Enum, width, height, border int, data []byte) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.CompressedTexImage2D(%v, %v, %v, %v, %v, %v, len(%d)) %v", target, level, internalformat, width, height, border, len(data), errstr)
	}()
	ctx.Call("compressedTexImage2D", target.c(), level, internalformat.c(), width, height, border, data)
}

func (ctx *context) CompressedTexSubImage2D(target Enum, level, xoffset, yoffset, width, height int, format Enum, data []byte) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.CompressedTexSubImage2D(%v, %v, %v, %v, %v, %v, %v, len(%d)) %v", target, level, xoffset, yoffset, width, height, format, len(data), errstr)
	}()
	ctx.Call("compressedTexSubImage2D", target.c(), level, xoffset, yoffset, width, height, format.c(), data)
}

func (ctx *context) CopyTexImage2D(target Enum, level int, internalformat Enum, x, y, width, height, border int) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.CopyTexImage2D(%v, %v, %v, %v, %v, %v, %v, %v) %v", target, level, internalformat, x, y, width, height, border, errstr)
	}()
	ctx.Call("copyTexImage2D", target.c(), level, internalformat.c(), x, y, width, height, border)
}

func (ctx *context) CopyTexSubImage2D(target Enum, level, xoffset, yoffset, x, y, width, height int) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.CopyTexSubImage2D(%v, %v, %v, %v, %v, %v, %v, %v) %v", target, level, xoffset, yoffset, x, y, width, height, errstr)
	}()
	ctx.Call("copyTexSubImage2D", target.c(), level, xoffset, yoffset, x, y, width, height)
}

func (ctx *context) CreateBuffer() (r0 Buffer) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.CreateBuffer() %v%v", r0, errstr)
	}()
	return Buffer{Value: ctx.Call("createBuffer")}
}

func (ctx *context) CreateFramebuffer() (r0 Framebuffer) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.CreateFramebuffer() %v%v", r0, errstr)
	}()
	return Framebuffer{Value: ctx.Call("createFramebuffer")}
}

func (ctx *context) CreateProgram() (r0 Program) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.CreateProgram() %v%v", r0, errstr)
	}()
	return Program{Value: ctx.Call("createProgram")}
}

func (ctx *context) CreateRenderbuffer() (r0 Renderbuffer) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.CreateRenderbuffer() %v%v", r0, errstr)
	}()
	return Renderbuffer{Value: ctx.Call("createRenderbuffer")}
}

func (ctx *context) CreateShader(ty Enum) (r0 Shader) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.CreateShader(%v) %v%v", ty, r0, errstr)
	}()
	return Shader{Value: ctx.Call("createShader")}
}

func (ctx *context) CreateTexture() (r0 Texture) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.CreateTexture() %v%v", r0, errstr)
	}()
	return Texture{Value: ctx.Call("createTexture")}
}

func (ctx *context) CreateVertexArray() (r0 VertexArray) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.CreateVertexArray() %v%v", r0, errstr)
	}()
	return VertexArray{Value: ctx.callWebGL2Compat("OES_vertex_array_object", "OES", "createVertexArray")}
}

func (ctx *context) CullFace(mode Enum) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.CullFace(%v) %v", mode, errstr)
	}()
	ctx.Call("cullFace", mode.c())
}

func (ctx *context) DeleteBuffer(v Buffer) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.DeleteBuffer(%v) %v", v, errstr)
	}()
	ctx.Call("deleteBuffer", v.c())
}

func (ctx *context) DeleteFramebuffer(v Framebuffer) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.DeleteFramebuffer(%v) %v", v, errstr)
	}()
	ctx.Call("deleteFramebuffer", v.c())
}

func (ctx *context) DeleteProgram(p Program) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.DeleteProgram(%v) %v", p, errstr)
	}()
	ctx.Call("deleteProgram", p.c())
}

func (ctx *context) DeleteRenderbuffer(v Renderbuffer) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.DeleteRenderbuffer(%v) %v", v, errstr)
	}()
	ctx.Call("deleteRenderbuffer", v.c())
}

func (ctx *context) DeleteShader(s Shader) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.DeleteShader(%v) %v", s, errstr)
	}()
	ctx.Call("deleteShader", s.c())
}

func (ctx *context) DeleteTexture(v Texture) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.DeleteTexture(%v) %v", v, errstr)
	}()
	ctx.Call("deleteTexture", v.c())
}

func (ctx *context) DeleteVertexArray(v VertexArray) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.DeleteVertexArray(%v) %v", v, errstr)
	}()
	ctx.callWebGL2Compat("OES_vertex_array_object", "OES", "deleteVertexArray", v.c())
}

func (ctx *context) DepthFunc(fn Enum) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.DepthFunc(%v) %v", fn, errstr)
	}()
	ctx.Call("depthFunc", fn.c())
}

func (ctx *context) DepthMask(flag bool) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.DepthMask(%v) %v", flag, errstr)
	}()
	ctx.Call("depthMask", flag)
}

func (ctx *context) DepthRangef(n, f float32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.DepthRangef(%v, %v) %v", n, f, errstr)
	}()
	ctx.Call("depthRange", n, f)
}

func (ctx *context) DetachShader(p Program, s Shader) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.DetachShader(%v, %v) %v", p, s, errstr)
	}()
	ctx.Call("detachShader", p.c(), s.c())
}

func (ctx *context) Disable(cap Enum) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.Disable(%v) %v", cap, errstr)
	}()
	ctx.Call("disable", cap.c())
}

func (ctx *context) DisableVertexAttribArray(a Attrib) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.DisableVertexAttribArray(%v) %v", a, errstr)
	}()
	ctx.Call("disableVertexAttribArray", a.c())
}

func (ctx *context) DrawArrays(mode Enum, first, count int) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.DrawArrays(%v, %v, %v) %v", mode, first, count, errstr)
	}()
	ctx.Call("drawArrays", mode.c(), first, count)
}

func (ctx *context) DrawElements(mode Enum, count int, ty Enum, offset int) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.DrawElements(%v, %v, %v, %v) %v", mode, count, ty, offset, errstr)
	}()
	ctx.Call("drawElements", mode.c(), count, ty.c(), offset)
}

func (ctx *context) Enable(cap Enum) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.Enable(%v) %v", cap, errstr)
	}()
	ctx.Call("enable", cap.c())
}

func (ctx *context) EnableVertexAttribArray(a Attrib) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.EnableVertexAttribArray(%v) %v", a, errstr)
	}()
	ctx.Call("enableVertexAttribArray", a.c())
}

func (ctx *context) Finish() {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.Finish() %v", errstr)
	}()
	ctx.Call("finish")
}

func (ctx *context) Flush() {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.Flush() %v", errstr)
	}()
	ctx.Call("flush")
}

func (ctx *context) FramebufferRenderbuffer(target, attachment, rbTarget Enum, rb Renderbuffer) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.FramebufferRenderbuffer(%v, %v, %v, %v) %v", target, attachment, rbTarget, rb, errstr)
	}()
	ctx.Call("framebufferRenderbuffer", target.c(), attachment.c(), rbTarget.c(), rb.c())
}

func (ctx *context) FramebufferTexture2D(target, attachment, texTarget Enum, t Texture, level int) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.FramebufferTexture2D(%v, %v, %v, %v, %v) %v", target, attachment, texTarget, t, level, errstr)
	}()
	ctx.Call("framebufferTexture2D", target.c(), attachment.c(), texTarget.c(), t.c(), level)
}

func (ctx *context) FrontFace(mode Enum) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.FrontFace(%v) %v", mode, errstr)
	}()
	ctx.Call("frontFace", mode.c())
}

func (ctx *context) GenerateMipmap(target Enum) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.GenerateMipmap(%v) %v", target, errstr)
	}()
	ctx.Call("generateMipmap", target.c())
}

func (ctx *context) GetActiveAttrib(p Program, index uint32) (name string, size int, ty Enum) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.GetActiveAttrib(%v, %v) (%v, %v, %v) %v", p, index, name, size, ty, errstr)
	}()
	info := ctx.Call("getActiveAttrib", p.c(), index)
	return info.Get("name").String(), info.Get("size").Int(), Enum(info.Get("type").Int())
}

func (ctx *context) GetActiveUniform(p Program, index uint32) (name string, size int, ty Enum) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.GetActiveUniform(%v, %v) (%v, %v, %v) %v", p, index, name, size, ty, errstr)
	}()
	info := ctx.Call("getActiveUniform", p.c(), index)
	return info.Get("name").String(), info.Get("size").Int(), Enum(info.Get("type").Int())
}

func (ctx *context) GetAttachedShaders(p Program) (r0 []Shader) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.GetAttachedShaders(%v) %v%v", p, r0, errstr)
	}()
	shaders := ctx.Call("getAttachedShaders", p.c())
	wrapped := make([]Shader, shaders.Length())
	for i := range wrapped {
		wrapped[i].Value = shaders.Index(i)
	}
	return wrapped
}

func (ctx *context) GetAttribLocation(p Program, name string) (r0 Attrib) {
	defer func() {
		errstr := ctx.errDrain()
		r0.name = name
		log.Printf("gl.GetAttribLocation(%v, %v) %v%v", p, name, r0, errstr)
	}()
	return Attrib{Value: uint(ctx.Call("getAttribLocation", p.c(), name).Int())}
}

func (ctx *context) GetBooleanv(dst []bool, pname Enum) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.GetBooleanv(%v, %v) %v", dst, pname, errstr)
	}()
	param := ctx.Call("getParameter", pname.c())
	for i := 0; i < param.Length(); i++ {
		dst[i] = param.Index(i).Bool()
	}
}

func (ctx *context) GetFloatv(dst []float32, pname Enum) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.GetFloatv(len(%d), %v) %v", len(dst), pname, errstr)
	}()
	param := ctx.Call("getParameter", pname.c())
	for i := 0; i < param.Length(); i++ {
		dst[i] = float32(param.Index(i).Float())
	}
}

func (ctx *context) GetIntegerv(dst []int32, pname Enum) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.GetIntegerv(%v, %v) %v", dst, pname, errstr)
	}()
	param := ctx.Call("getParameter", pname.c())
	for i := 0; i < param.Length(); i++ {
		dst[i] = int32(param.Index(i).Int())
	}
}

func (ctx *context) GetInteger(pname Enum) (r0 int) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.GetInteger(%v) %v%v", pname, r0, errstr)
	}()
	return ctx.Call("getParameter", pname.c()).Int()
}

func (ctx *context) GetBufferParameteri(target, value Enum) (r0 int) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.GetBufferParameteri(%v, %v) %v%v", target, value, r0, errstr)
	}()
	return ctx.Call("getBufferParameter", target.c(), value.c()).Int()
}

func (ctx *context) GetError() (r0 Enum) {
	return Enum(ctx.Call("getError").Int())
}

func (ctx *context) GetFramebufferAttachmentParameteri(target, attachment, pname Enum) (r0 int) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.GetFramebufferAttachmentParameteri(%v, %v, %v) %v%v", target, attachment, pname, r0, errstr)
	}()
	return ctx.Call("getFramebufferAttachmentParameter", target.c(), attachment.c(), pname.c()).Int()
}

func (ctx *context) GetProgrami(p Program, pname Enum) (r0 int) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.GetProgrami(%v, %v) %v%v", p, pname, r0, errstr)
	}()
	return ctx.Call("getProgramParameter", p.c(), pname.c()).Int()
}

func (ctx *context) GetProgramInfoLog(p Program) (r0 string) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.GetProgramInfoLog(%v) %v%v", p, r0, errstr)
	}()
	return ctx.Call("getProgramInfoLog", p.c()).String()
}

func (ctx *context) GetRenderbufferParameteri(target, pname Enum) (r0 int) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.GetRenderbufferParameteri(%v, %v) %v%v", target, pname, r0, errstr)
	}()
	return ctx.Call("getRenderbufferParameter", target.c(), pname.c()).Int()
}

func (ctx *context) GetShaderi(s Shader, pname Enum) (r0 int) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.GetShaderi(%v, %v) %v%v", s, pname, r0, errstr)
	}()
	return ctx.Call("getShaderParameter", s.c(), pname.c()).Int()
}

func (ctx *context) GetShaderInfoLog(s Shader) (r0 string) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.GetShaderInfoLog(%v) %v%v", s, r0, errstr)
	}()
	return ctx.Call("getShaderInfoLog", s.c()).String()
}

func (ctx *context) GetShaderPrecisionFormat(shadertype, precisiontype Enum) (rangeLow, rangeHigh, precision int) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.GetShaderPrecisionFormat(%v, %v) (%v, %v, %v) %v", shadertype, precisiontype, rangeLow, rangeHigh, precision, errstr)
	}()
	prec := ctx.Call("getShaderPrecisionFormat", shadertype.c(), precisiontype.c())
	return prec.Get("rangeMin").Int(), prec.Get("rangeMax").Int(), prec.Get("precision").Int()
}

func (ctx *context) GetShaderSource(s Shader) (r0 string) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.GetShaderSource(%v) %v%v", s, r0, errstr)
	}()
	return ctx.Call("getShaderSource", s.c()).String()
}

func (ctx *context) GetString(pname Enum) (r0 string) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.GetString(%v) %v%v", pname, r0, errstr)
	}()
	return ctx.Call("getParameter", pname.c()).String()
}

func (ctx *context) GetTexParameterfv(dst []float32, target, pname Enum) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.GetTexParameterfv(len(%d), %v, %v) %v", len(dst), target, pname, errstr)
	}()
	param := ctx.Call("getTexParameter", target.c(), pname.c())
	for i := 0; i < param.Length(); i++ {
		dst[i] = float32(param.Index(i).Float())
	}
}

func (ctx *context) GetTexParameteriv(dst []int32, target, pname Enum) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.GetTexParameteriv(%v, %v, %v) %v", dst, target, pname, errstr)
	}()
	param := ctx.Call("getTexParameter", target.c(), pname.c())
	for i := 0; i < param.Length(); i++ {
		dst[i] = int32(param.Index(i).Int())
	}
}

func (ctx *context) GetUniformfv(dst []float32, src Uniform, p Program) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.GetUniformfv(len(%d), %v, %v) %v", len(dst), src, p, errstr)
	}()
	param := ctx.Call("getUniform", p.c(), src.c())
	for i := 0; i < param.Length(); i++ {
		dst[i] = float32(param.Index(i).Float())
	}
}

func (ctx *context) GetUniformiv(dst []int32, src Uniform, p Program) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.GetUniformiv(%v, %v, %v) %v", dst, src, p, errstr)
	}()
	param := ctx.Call("getUniform", p.c(), src.c())
	for i := 0; i < param.Length(); i++ {
		dst[i] = int32(param.Index(i).Int())
	}
}

func (ctx *context) GetUniformLocation(p Program, name string) (r0 Uniform) {
	defer func() {
		errstr := ctx.errDrain()
		r0.name = name
		log.Printf("gl.GetUniformLocation(%v, %v) %v%v", p, name, r0, errstr)
	}()
	return Uniform{Value: ctx.Call("getUniformLocation", p.c(), name)}
}

func (ctx *context) GetVertexAttribf(src Attrib, pname Enum) (r0 float32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.GetVertexAttribf(%v, %v) %v%v", src, pname, r0, errstr)
	}()
	var params [1]float32
	ctx.GetVertexAttribfv(params[:], src, pname)
	return params[0]
}

func (ctx *context) GetVertexAttribfv(dst []float32, src Attrib, pname Enum) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.GetVertexAttribfv(len(%d), %v, %v) %v", len(dst), src, pname, errstr)
	}()
	param := ctx.Call("getVertexAttrib", src.c(), pname.c())
	if param.Get("length") == js.Undefined {
		dst[0] = float32(param.Float())
		return
	}
	for i := 0; i < param.Length(); i++ {
		dst[i] = float32(param.Index(i).Float())
	}
}

func (ctx *context) GetVertexAttribi(src Attrib, pname Enum) (r0 int32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.GetVertexAttribi(%v, %v) %v%v", src, pname, r0, errstr)
	}()
	var params [1]int32
	ctx.GetVertexAttribiv(params[:], src, pname)
	return params[0]
}

func (ctx *context) GetVertexAttribiv(dst []int32, src Attrib, pname Enum) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.GetVertexAttribiv(%v, %v, %v) %v", dst, src, pname, errstr)
	}()
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
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.Hint(%v, %v) %v", target, mode, errstr)
	}()
	ctx.Call("hint", target.c(), mode.c())
}

func (ctx *context) IsBuffer(b Buffer) (r0 bool) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.IsBuffer(%v) %v%v", b, r0, errstr)
	}()
	return b.Value != nil && ctx.Call("isBuffer", b.c()).Bool()
}

func (ctx *context) IsEnabled(cap Enum) (r0 bool) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.IsEnabled(%v) %v%v", cap, r0, errstr)
	}()
	return ctx.Call("isEnabled", cap.c()).Bool()
}

func (ctx *context) IsFramebuffer(fb Framebuffer) (r0 bool) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.IsFramebuffer(%v) %v%v", fb, r0, errstr)
	}()
	return fb.Value != nil && ctx.Call("isFramebuffer", fb.c()).Bool()
}

func (ctx *context) IsProgram(p Program) (r0 bool) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.IsProgram(%v) %v%v", p, r0, errstr)
	}()
	return p.Value != nil && ctx.Call("isProgram", p.c()).Bool()
}

func (ctx *context) IsRenderbuffer(rb Renderbuffer) (r0 bool) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.IsRenderbuffer(%v) %v%v", rb, r0, errstr)
	}()
	return rb.Value != nil && ctx.Call("isRenderbuffer", rb.c()).Bool()
}

func (ctx *context) IsShader(s Shader) (r0 bool) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.IsShader(%v) %v%v", s, r0, errstr)
	}()
	return s.Value != nil && ctx.Call("isShader", s.c()).Bool()
}

func (ctx *context) IsTexture(t Texture) (r0 bool) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.IsTexture(%v) %v%v", t, r0, errstr)
	}()
	return t.Value != nil && ctx.Call("isTexture", t.c()).Bool()
}

func (ctx *context) LineWidth(width float32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.LineWidth(%v) %v", width, errstr)
	}()
	ctx.Call("lineWidth", width)
}

func (ctx *context) LinkProgram(p Program) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.LinkProgram(%v) %v", p, errstr)
	}()
	ctx.Call("linkProgram", p.c())
}

func (ctx *context) PixelStorei(pname Enum, param int32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.PixelStorei(%v, %v) %v", pname, param, errstr)
	}()
	ctx.Call("pixelStorei", pname.c(), param)
}

func (ctx *context) PolygonOffset(factor, units float32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.PolygonOffset(%v, %v) %v", factor, units, errstr)
	}()
	ctx.Call("polygonOffset", factor, units)
}

func (ctx *context) ReadPixels(dst []byte, x, y, width, height int, format, ty Enum) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.ReadPixels(len(%d), %v, %v, %v, %v, %v, %v) %v", len(dst), x, y, width, height, format, ty, errstr)
	}()
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
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.ReleaseShaderCompiler() %v", errstr)
	}()
}

func (ctx *context) RenderbufferStorage(target, internalFormat Enum, width, height int) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.RenderbufferStorage(%v, %v, %v, %v) %v", target, internalFormat, width, height, errstr)
	}()
	ctx.Call("renderbufferStorage", target.c(), internalFormat.c(), width, height)
}

func (ctx *context) SampleCoverage(value float32, invert bool) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.SampleCoverage(%v, %v) %v", value, invert, errstr)
	}()
	ctx.Call("sampleCoverage", value, invert)
}

func (ctx *context) Scissor(x, y, width, height int32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.Scissor(%v, %v, %v, %v) %v", x, y, width, height, errstr)
	}()
	ctx.Call("scissor", x, y, width, height)
}

func (ctx *context) ShaderSource(s Shader, src string) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.ShaderSource(%v, %v) %v", s, src, errstr)
	}()
	ctx.Call("shaderSource", s.c(), src)
}

func (ctx *context) StencilFunc(fn Enum, ref int, mask uint32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.StencilFunc(%v, %v, %v) %v", fn, ref, mask, errstr)
	}()
	ctx.Call("stencilFunc", fn.c(), ref, mask)
}

func (ctx *context) StencilFuncSeparate(face, fn Enum, ref int, mask uint32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.StencilFuncSeparate(%v, %v, %v, %v) %v", face, fn, ref, mask, errstr)
	}()
	ctx.Call("stencilFuncSeparate", face.c(), fn.c(), ref, mask)
}

func (ctx *context) StencilMask(mask uint32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.StencilMask(%v) %v", mask, errstr)
	}()
	ctx.Call("stencilMask", mask)
}

func (ctx *context) StencilMaskSeparate(face Enum, mask uint32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.StencilMaskSeparate(%v, %v) %v", face, mask, errstr)
	}()
	ctx.Call("stencilMaskSeparate", face.c(), mask)
}

func (ctx *context) StencilOp(fail, zfail, zpass Enum) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.StencilOp(%v, %v, %v) %v", fail, zfail, zpass, errstr)
	}()
	ctx.Call("stencilOp", fail.c(), zfail.c(), zpass.c())
}

func (ctx *context) StencilOpSeparate(face, sfail, dpfail, dppass Enum) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.StencilOpSeparate(%v, %v, %v, %v) %v", face, sfail, dpfail, dppass, errstr)
	}()
	ctx.Call("stencilOpSeparate", face.c(), sfail.c(), dpfail.c(), dppass.c())
}

func (ctx *context) TexImage2D(target Enum, level int, width, height int, format Enum, ty Enum, data []byte) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.TexImage2D(%v, %v, %v, %v, %v, %v, len(%d)) %v", target, level, width, height, format, ty, len(data), errstr)
	}()
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
	ctx.Call("texImage2D", target.c(), level, format.c(), width, height, 0, format.c(), ty.c(), dataView)
}

func (ctx *context) TexSubImage2D(target Enum, level int, x, y, width, height int, format, ty Enum, data []byte) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.TexSubImage2D(%v, %v, %v, %v, %v, %v, %v, %v, len(%d)) %v", target, level, x, y, width, height, format, ty, len(data), errstr)
	}()
	dataView := js.NewArrayBuffer(data)
	switch ty {
	case UNSIGNED_BYTE:
		dataView = js.Global.Get("Uint8Array").New(dataView)
	case UNSIGNED_SHORT_5_6_5, UNSIGNED_SHORT_4_4_4_4, UNSIGNED_SHORT_5_5_5_1, HALF_FLOAT:
		dataView = js.Global.Get("Uint16Array").New(dataView)
	case FLOAT:
		dataView = js.Global.Get("Float32Array").New(dataView)
	}
	ctx.Call("texSubImage2D", target.c(), level, x, y, width, height, format.c(), ty.c(), dataView)
}

func (ctx *context) TexParameterf(target, pname Enum, param float32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.TexParameterf(%v, %v, %v) %v", target, pname, param, errstr)
	}()
	ctx.Call("texParameterf", target.c(), pname.c(), param)
}

func (ctx *context) TexParameterfv(target, pname Enum, params []float32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.TexParameterfv(%v, %v, len(%d)) %v", target, pname, len(params), errstr)
	}()
	ctx.Call("texParameterfv", target.c(), pname.c(), params)
}

func (ctx *context) TexParameteri(target, pname Enum, param int) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.TexParameteri(%v, %v, %v) %v", target, pname, param, errstr)
	}()
	ctx.Call("texParameteri", target.c(), pname.c(), param)
}

func (ctx *context) TexParameteriv(target, pname Enum, params []int32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.TexParameteriv(%v, %v, %v) %v", target, pname, params, errstr)
	}()
	ctx.Call("texParameteriv", target.c(), pname.c(), params)
}

func (ctx *context) Uniform1f(dst Uniform, v float32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.Uniform1f(%v, %v) %v", dst, v, errstr)
	}()
	ctx.Call("uniform1f", dst.c(), v)
}

func (ctx *context) Uniform1fv(dst Uniform, src []float32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.Uniform1fv(%v, len(%d)) %v", dst, len(src), errstr)
	}()
	ctx.Call("uniform1fv", dst.c(), src)
}

func (ctx *context) Uniform1i(dst Uniform, v int) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.Uniform1i(%v, %v) %v", dst, v, errstr)
	}()
	ctx.Call("uniform1i", dst.c(), v)
}

func (ctx *context) Uniform1iv(dst Uniform, src []int32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.Uniform1iv(%v, %v) %v", dst, src, errstr)
	}()
	ctx.Call("uniform1iv", dst.c(), src)
}

func (ctx *context) Uniform2f(dst Uniform, v0, v1 float32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.Uniform2f(%v, %v, %v) %v", dst, v0, v1, errstr)
	}()
	ctx.Call("uniform2f", dst.c(), v0, v1)
}

func (ctx *context) Uniform2fv(dst Uniform, src []float32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.Uniform2fv(%v, len(%d)) %v", dst, len(src), errstr)
	}()
	ctx.Call("uniform2fv", dst.c(), src)
}

func (ctx *context) Uniform2i(dst Uniform, v0, v1 int) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.Uniform2i(%v, %v, %v) %v", dst, v0, v1, errstr)
	}()
	ctx.Call("uniform2i", dst.c(), v0, v1)
}

func (ctx *context) Uniform2iv(dst Uniform, src []int32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.Uniform2iv(%v, %v) %v", dst, src, errstr)
	}()
	ctx.Call("uniform2iv", dst.c(), src)
}

func (ctx *context) Uniform3f(dst Uniform, v0, v1, v2 float32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.Uniform3f(%v, %v, %v, %v) %v", dst, v0, v1, v2, errstr)
	}()
	ctx.Call("uniform3f", dst.c(), v0, v1, v2)
}

func (ctx *context) Uniform3fv(dst Uniform, src []float32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.Uniform3fv(%v, len(%d)) %v", dst, len(src), errstr)
	}()
	ctx.Call("uniform3fv", dst.c(), src)
}

func (ctx *context) Uniform3i(dst Uniform, v0, v1, v2 int32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.Uniform3i(%v, %v, %v, %v) %v", dst, v0, v1, v2, errstr)
	}()
	ctx.Call("uniform3i", dst.c(), v0, v1, v2)
}

func (ctx *context) Uniform3iv(dst Uniform, src []int32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.Uniform3iv(%v, %v) %v", dst, src, errstr)
	}()
	ctx.Call("uniform3iv", dst.c(), src)
}

func (ctx *context) Uniform4f(dst Uniform, v0, v1, v2, v3 float32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.Uniform4f(%v, %v, %v, %v, %v) %v", dst, v0, v1, v2, v3, errstr)
	}()
	ctx.Call("uniform4f", dst.c(), v0, v1, v2, v3)
}

func (ctx *context) Uniform4fv(dst Uniform, src []float32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.Uniform4fv(%v, len(%d)) %v", dst, len(src), errstr)
	}()
	ctx.Call("uniform4fv", dst.c(), src)
}

func (ctx *context) Uniform4i(dst Uniform, v0, v1, v2, v3 int32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.Uniform4i(%v, %v, %v, %v, %v) %v", dst, v0, v1, v2, v3, errstr)
	}()
	ctx.Call("uniform4i", dst.c(), v0, v1, v2, v3)
}

func (ctx *context) Uniform4iv(dst Uniform, src []int32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.Uniform4iv(%v, %v) %v", dst, src, errstr)
	}()
	ctx.Call("uniform4iv", dst.c(), src)
}

func (ctx *context) UniformMatrix2fv(dst Uniform, src []float32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.UniformMatrix2fv(%v, len(%d)) %v", dst, len(src), errstr)
	}()
	ctx.Call("uniformMatrix2fv", dst.c(), src)
}

func (ctx *context) UniformMatrix3fv(dst Uniform, src []float32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.UniformMatrix3fv(%v, len(%d)) %v", dst, len(src), errstr)
	}()
	ctx.Call("uniformMatrix3fv", dst.c(), src)
}

func (ctx *context) UniformMatrix4fv(dst Uniform, src []float32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.UniformMatrix4fv(%v, len(%d)) %v", dst, len(src), errstr)
	}()
	ctx.Call("uniformMatrix4fv", dst.c(), src)
}

func (ctx *context) UseProgram(p Program) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.UseProgram(%v) %v", p, errstr)
	}()
	ctx.Call("useProgram", p.c())
}

func (ctx *context) ValidateProgram(p Program) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.ValidateProgram(%v) %v", p, errstr)
	}()
	ctx.Call("validateProgram", p.c())
}

func (ctx *context) VertexAttrib1f(dst Attrib, x float32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.VertexAttrib1f(%v, %v) %v", dst, x, errstr)
	}()
	ctx.Call("vertexAttrib1f", dst.c(), x)
}

func (ctx *context) VertexAttrib1fv(dst Attrib, src []float32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.VertexAttrib1fv(%v, len(%d)) %v", dst, len(src), errstr)
	}()
	ctx.Call("vertexAttrib1fv", dst.c(), src)
}

func (ctx *context) VertexAttrib2f(dst Attrib, x, y float32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.VertexAttrib2f(%v, %v, %v) %v", dst, x, y, errstr)
	}()
	ctx.Call("vertexAttrib2f", dst.c(), x, y)
}

func (ctx *context) VertexAttrib2fv(dst Attrib, src []float32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.VertexAttrib2fv(%v, len(%d)) %v", dst, len(src), errstr)
	}()
	ctx.Call("vertexAttrib2fv", dst.c(), src)
}

func (ctx *context) VertexAttrib3f(dst Attrib, x, y, z float32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.VertexAttrib3f(%v, %v, %v, %v) %v", dst, x, y, z, errstr)
	}()
	ctx.Call("vertexAttrib3f", dst.c(), x, y, z)
}

func (ctx *context) VertexAttrib3fv(dst Attrib, src []float32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.VertexAttrib3fv(%v, len(%d)) %v", dst, len(src), errstr)
	}()
	ctx.Call("vertexAttrib3fv", dst.c(), src)
}

func (ctx *context) VertexAttrib4f(dst Attrib, x, y, z, w float32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.VertexAttrib4f(%v, %v, %v, %v, %v) %v", dst, x, y, z, w, errstr)
	}()
	ctx.Call("vertexAttrib4f", dst.c(), x, y, z, w)
}

func (ctx *context) VertexAttrib4fv(dst Attrib, src []float32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.VertexAttrib4fv(%v, len(%d)) %v", dst, len(src), errstr)
	}()
	ctx.Call("vertexAttrib4fv", dst.c(), src)
}

func (ctx *context) VertexAttribPointer(dst Attrib, size int, ty Enum, normalized bool, stride, offset int) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.VertexAttribPointer(%v, %v, %v, %v, %v, %v) %v", dst, size, ty, normalized, stride, offset, errstr)
	}()
	ctx.Call("vertexAttribPointer", dst.c(), size, ty.c(), normalized, stride, offset)
}

func (ctx *context) Viewport(x, y, width, height int) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.Viewport(%v, %v, %v, %v) %v", x, y, width, height, errstr)
	}()
	ctx.Call("viewport", x, y, width, height)
}

func (ctx context3) UniformMatrix2x3fv(dst Uniform, src []float32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.UniformMatrix2x3fv(%v, len(%d)) %v", dst, len(src), errstr)
	}()
	ctx.Call("uniformMatrix2x3fv", dst.c(), src)
}

func (ctx context3) UniformMatrix3x2fv(dst Uniform, src []float32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.UniformMatrix3x2fv(%v, len(%d)) %v", dst, len(src), errstr)
	}()
	ctx.Call("uniformMatrix3x2fv", dst.c(), src)
}

func (ctx context3) UniformMatrix2x4fv(dst Uniform, src []float32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.UniformMatrix2x4fv(%v, len(%d)) %v", dst, len(src), errstr)
	}()
	ctx.Call("uniformMatrix2x4fv", dst.c(), src)
}

func (ctx context3) UniformMatrix4x2fv(dst Uniform, src []float32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.UniformMatrix4x2fv(%v, len(%d)) %v", dst, len(src), errstr)
	}()
	ctx.Call("uniformMatrix4x2fv", dst.c(), src)
}

func (ctx context3) UniformMatrix3x4fv(dst Uniform, src []float32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.UniformMatrix3x4fv(%v, len(%d)) %v", dst, len(src), errstr)
	}()
	ctx.Call("uniformMatrix3x4fv", dst.c(), src)
}

func (ctx context3) UniformMatrix4x3fv(dst Uniform, src []float32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.UniformMatrix4x3fv(%v, len(%d)) %v", dst, len(src), errstr)
	}()
	ctx.Call("uniformMatrix4x3fv", dst.c(), src)
}

func (ctx context3) BlitFramebuffer(srcX0, srcY0, srcX1, srcY1, dstX0, dstY0, dstX1, dstY1 int, mask uint, filter Enum) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.BlitFramebuffer(%v, %v, %v, %v, %v, %v, %v, %v, %v, %v) %v", srcX0, srcY0, srcX1, srcY1, dstX0, dstY0, dstX1, dstY1, mask, filter, errstr)
	}()
	ctx.Call("blitFramebuffer", srcX0, srcY0, srcX1, srcY1, dstX0, dstY0, dstX1, dstY1, mask, filter.c())
}

func (ctx context3) Uniform1ui(dst Uniform, v uint32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.Uniform1ui(%v, %v) %v", dst, v, errstr)
	}()
	ctx.Call("uniform1ui", dst.c(), v)
}

func (ctx context3) Uniform2ui(dst Uniform, v0, v1 uint32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.Uniform2ui(%v, %v, %v) %v", dst, v0, v1, errstr)
	}()
	ctx.Call("uniform2ui", dst.c(), v0, v1)
}

func (ctx context3) Uniform3ui(dst Uniform, v0, v1, v2 uint) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.Uniform3ui(%v, %v, %v, %v) %v", dst, v0, v1, v2, errstr)
	}()
	ctx.Call("uniform3ui", dst.c(), v0, v1, v2)
}

func (ctx context3) Uniform4ui(dst Uniform, v0, v1, v2, v3 uint32) {
	defer func() {
		errstr := ctx.errDrain()
		log.Printf("gl.Uniform4ui(%v, %v, %v, %v, %v) %v", dst, v0, v1, v2, v3, errstr)
	}()
	ctx.Call("uniform4ui", dst.c(), v0, v1, v2, v3)
}
