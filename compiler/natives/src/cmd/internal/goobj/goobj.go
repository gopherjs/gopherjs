//go:build js

package goobj

//gopherjs:replace Used unsafeheader.String
func toString(b []byte) string {
	return string(b)
}
