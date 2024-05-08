//go:build js
// +build js

package slicereader

func toString(b []byte) string {
	if len(b) == 0 {
		return ``
	}
	// Overwritten to avoid `unsafe.String`
	return string(b)
}
