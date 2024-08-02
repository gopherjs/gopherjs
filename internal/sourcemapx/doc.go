// Package sourcemapx contains utilities for passing source map information
// around, intended to work with github.com/neelance/sourcemap.
//
// GopherJS code generator outputs hints about correspondence between the
// generated code and original sources inline. Such hints are marked by the
// special `\b` (0x08) magic byte, followed by a variable-length sequence of
// bytes, which can be extracted from the byte slice using ReadHint() function.
//
// '\b' was chosen as a magic symbol because it would never occur unescaped in
// the generated code, other than when explicitly inserted by the source mapping
// hint. See Hint type documentation for the details of the encoded format.
//
// The hinting mechanism is designed to be extensible, the Hint type able to
// wrap different types containing different information:
//
//   - go/token.Pos indicates position in the original source the current
//     location in the generated code corresponds to.
//   - Identifier maps a JS identifier to the original Go identifier it
//     represents.
//
// More types may be added in future if necessary.
//
// Filter type is used to extract the hints from the written code stream and
// pass them into source map generator. It also ensures that the encoded inline
// hints don't make it into the final output, since they are not valid JS.
package sourcemapx
