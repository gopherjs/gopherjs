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

func (f *minifier) Write(b []byte) (int, error) {
	if !f.minify {
		return f.out.Write(b)
	}

	consumed := 0
	writePending := func(stop int) error {
		if consumed >= stop {
			return nil
		}
		n2, err := f.out.Write(b[consumed:stop])
		consumed += n2
		return err
	}

	var previous byte
	count := len(b)
	for i := 0; i < count; i++ {
		switch b[i] {
		case ' ', '\t', '\n':
			if skipSpace(previous, b[i:]) {
				writePending(i)
				consumed++
				// don't update previous so that we can skip all but one
				// multiple spaces between words
				continue
			}
		case '"':
			if len := stringLength(b); len > 0 {
				i += len - 1
			}
		case '/':
			if skip := commentLength(b); skip > 0 {
				writePending(i)
				i += skip - 1
				consumed += skip
			}
		}
		previous = b[i]
	}
	writePending(count)
	return consumed, nil
}

func needsSpace(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '$'
}

func skipSpace(prev byte, p []byte) bool {
	if len(p) < 2 {
		return false
	}
	next := p[1]
	if prev == '-' && next == '-' {
		return false
	}
	return !needsSpace(prev) || !needsSpace(next)
}

func stringLength(p []byte) int {
	count := len(p)
	if count < 2 || p[0] != '"' {
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
	if len(p) < 2 || p[0] != '/' {
		return 0
	}
	switch p[1] {
	case '/':
		if i := bytes.IndexByte(p[2:], '\n'); i >= 0 {
			return i + 3
		}
		// end of input without newline
		return len(p)
	case '*':
		if i := bytes.Index(p[2:], []byte("*/")); i >= 0 {
			return i + 4
		}
		// end of input without closing comment
		return len(p)
	default:
		// not a comment
		return 0
	}
}
