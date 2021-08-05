//go:build js
// +build js

package os

// WriteString copied from Go 1.16, before it was made more peformant, and unsafe.
func (f *File) WriteString(s string) (n int, err error) {
	return f.Write([]byte(s))
}
