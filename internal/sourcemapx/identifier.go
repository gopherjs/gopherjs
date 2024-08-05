package sourcemapx

import (
	"fmt"
	"go/token"
	"strings"
)

// Identifier represents a generated code identifier with a the associated
// original identifier information, which can be used to produce a source map.
//
// This allows us to map a JS function or variable name back to the original Go
// symbol name.
type Identifier struct {
	Name         string    // Identifier to use in the generated code.
	OriginalName string    // Original identifier name.
	OriginalPos  token.Pos // Original identifier position.
}

// String returns generated code identifier name.
func (i Identifier) String() string {
	return i.Name
}

// EncodeHint returns a string with an encoded source map hint. The hint can be
// inserted into the generated code to be later extracted by the SourceMapFilter
// to produce a source map.
func (i Identifier) EncodeHint() string {
	buf := &strings.Builder{}
	h := Hint{}
	if err := h.Pack(i); err != nil {
		panic(fmt.Errorf("failed to pack identifier source map hint: %w", err))
	}
	if _, err := h.WriteTo(buf); err != nil {
		panic(fmt.Errorf("failed to write source map hint into a buffer: %w", err))
	}
	return buf.String()
}
