package sourceWriter

import (
	"bytes"
	"io"
)

var _ io.Writer = (*minifier)(nil)

type minifier struct {
	out    io.Writer
	minify bool
}

func newMinifier(w io.Writer, minify bool) *minifier {
	return &minifier{
		out:    w,
		minify: minify,
	}
}

func (f *minifier) SetMinify(minify bool) {
	f.minify = minify
}

func (f *minifier) Minify() bool {
	return f.minify
}

func (f *minifier) Write(b []byte) (n int, err error) {
	if !f.minify {
		return f.out.Write(b)
	}

	n = 0
	var n2 int
	writePending := func(stop int) bool {
		if n < stop {
			n2, err = f.out.Write(b[n:stop])
			n += n2
			if err != nil {
				return true
			}
		}
		return false
	}

	var previous byte
	count := len(b)
	for i := 0; i < count; i++ {
		cur := b[i]
		switch cur {
		case ' ', '\t', '\n':
			if skipSpace(previous, b[i:]) {
				if writePending(i) {
					return n, err
				}
				n++
				// don't update previous so that we can skip all but one
				// multiple spaces between words
				continue
			}
		case '"':
			if len := stringLength(b[i:]); len > 0 {
				i += len - 1
			}
		case '/':
			if skip := commentLength(b[i:]); skip > 0 {
				if writePending(i) {
					return n, err
				}
				i += skip - 1
				n += skip
				// don't update previous so that we treat a comment like
				// it didn't exist when skipping spaces around the comment
				continue
			}
		}
		previous = cur
	}
	writePending(count)
	return n, err
}

func needsSpace(c byte) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '_' || c == '$'
}

func skipSpace(prev byte, p []byte) bool {
	if len(p) < 2 || prev == 0 {
		// the space is a leading space or a tailing space so skip it
		return true
	}
	next := p[1]
	if prev == '-' && next == '-' {
		// the space is between two dashes, so keep it
		return false
	}
	return !needsSpace(prev) || !needsSpace(next)
}

func stringLength(p []byte) int {
	count := len(p)
	if count < 2 {
		return 0
	}
	for i := 1; i < count; i++ {
		switch p[i] {
		case '"':
			return i + 1 // include the closing quote
		case '\\':
			i++ // skip the escaped character
		}
	}
	return count // end of input without closing quote
}

func commentLength(p []byte) int {
	if len(p) < 2 {
		return 0
	}
	switch p[1] {
	case '*':
		if i := bytes.Index(p[2:], []byte("*/")); i >= 0 {
			return i + 4
		}
		// end of input without closing comment
		return len(p)
	case '/':
		if i := bytes.IndexByte(p[2:], '\n'); i >= 0 {
			// exclude the newline so that it can be skipped or
			// kept as whitespace
			return i + 2
		}
		// end of input without newline
		return len(p)
	default:
		// not a comment
		return 0
	}
}
