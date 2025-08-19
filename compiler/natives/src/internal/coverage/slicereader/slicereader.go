//go:build js

package slicereader

// Overwritten to avoid `unsafe.String`
func toString(b []byte) string {
	return string(b)
}
