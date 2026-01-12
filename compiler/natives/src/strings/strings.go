//go:build js

package strings

import (
	"unicode/utf8"

	"github.com/gopherjs/gopherjs/js"
)

func IndexByte(s string, c byte) int {
	return js.InternalObject(s).Call("indexOf", js.Global.Get("String").Call("fromCharCode", c)).Int()
}

func Index(s, sep string) int {
	return js.InternalObject(s).Call("indexOf", js.InternalObject(sep)).Int()
}

func LastIndex(s, sep string) int {
	return js.InternalObject(s).Call("lastIndexOf", js.InternalObject(sep)).Int()
}

func Count(s, sep string) int {
	n := 0
	// special cases
	switch {
	case len(sep) == 0:
		return utf8.RuneCountInString(s) + 1
	case len(sep) > len(s):
		return 0
	case len(sep) == len(s):
		if sep == s {
			return 1
		}
		return 0
	}

	for {
		pos := Index(s, sep)
		if pos == -1 {
			break
		}
		n++
		s = s[pos+len(sep):]
	}
	return n
}

func (b *Builder) String() string {
	// Upstream Builder.String relies on package unsafe. We can't do that.
	// TODO: It's possible that the entire strings.Builder API can be implemented
	//       more efficiently for GOARCH=js specifically (avoid using []byte, instead
	//       use a String directly; or some JavaScript string builder API if one exists).
	//       But this is more work, defer doing it until there's a need shown via profiling,
	//       and there are benchmarks available (see https://github.com/golang/go/issues/18990#issuecomment-352068533).
	return string(b.buf)
}

func (b *Builder) copyCheck() {
	if b.addr == nil {
		// Upstream copyCheck uses noescape, which performs unsafe.Pointer manipulation.
		// We can't do that, so skip it. See https://github.com/golang/go/commit/484586c81a0196e42ac52f651bc56017ca454280.
		b.addr = b
	} else if b.addr != b {
		panic("strings: illegal use of non-zero Builder copied by value")
	}
}

func Clone(s string) string {
	// Since in the JavaScript runtime we don't have access the string's
	// baking memory, we let the engine's garbage collector deal with substring
	// memory overheads and simply return the string as-is.
	return s
}

// Repeat is the go1.19 implementation of strings.Repeat.
//
// In the go1.20 implementation, the function was changed to use chunks that
// are 8KB in size to improve speed and cache access. This change is faster
// when running native Go code. However, for GopherJS, the change is much
// slower than the go1.19 implementation.
//
// The go1.20 change made tests like encoding/pem TestCVE202224675 take
// significantly longer to run for GopherJS.
// go1.19 concatenates 24 times and the test takes about 8 seconds.
// go1.20 concatenates about 15000 times and can take over a hour.
//
// We can't use `js.InternalObject(s).Call("repeat", count).String()`
// because JS performs additional UTF-8 escapes meaning tests like
// hash/adler32 TestGolden will fail because the wrong input is created.
func Repeat(s string, count int) string {
	if count == 0 {
		return ""
	}

	// Since we cannot return an error on overflow,
	// we should panic if the repeat will generate
	// an overflow.
	// See Issue golang.org/issue/16237
	if count < 0 {
		panic("strings: negative Repeat count")
	} else if len(s)*count/count != len(s) {
		panic("strings: Repeat count causes overflow")
	}

	n := len(s) * count
	var b Builder
	b.Grow(n)
	b.WriteString(s)
	for b.Len() < n {
		if b.Len() <= n/2 {
			b.WriteString(b.String())
		} else {
			b.WriteString(b.String()[:n-b.Len()])
			break
		}
	}
	return b.String()
}
