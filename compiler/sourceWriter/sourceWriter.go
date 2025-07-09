package sourceWriter

import (
	"fmt"
	"go/token"
	"io"
)

var _ io.Writer = (*SourceWriter)(nil)
var _ io.StringWriter = (*SourceWriter)(nil)

type SourceWriter struct {
	out      io.Writer
	filter   *sourceMapFilter
	minifier *minifier
}

func New(out io.Writer, mappingCallback MappingCallbackHandle, minify bool) *SourceWriter {
	// Perform the filtering prior to minification.
	// in --> filter --> minifier --> out
	minifier := newMinifier(out, minify)
	filter := newSourceMapFilter(minifier, mappingCallback)
	return &SourceWriter{
		out:      out,
		filter:   filter,
		minifier: minifier,
	}
}

func (f *SourceWriter) SetFileSet(fileSet *token.FileSet) {
	f.filter.SetFileSet(fileSet)
}

func (f *SourceWriter) SetMinify(minify bool) {
	f.minifier.SetMinify(minify)
}

func (f *SourceWriter) Minify() bool {
	return f.minifier.Minify()
}

func (f *SourceWriter) WriteUnminified(s string) (int, error) {
	if f.Minify() {
		f.SetMinify(false)
		defer f.SetMinify(true)
	}
	return f.WriteString(s)
}

func (f *SourceWriter) WriteF(format string, args ...any) (int, error) {
	return fmt.Fprintf(f.out, format, args...)
}

func (f *SourceWriter) WriteString(s string) (int, error) {
	return f.Write([]byte(s))
}

func (f *SourceWriter) Write(p []byte) (n int, err error) {
	return f.filter.Write(p)
}
