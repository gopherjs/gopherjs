//go:build js
// +build js

package time

// replaced `parseRFC3339[string]` for go1.20 temporarily without generics.
var ParseRFC3339 = func(s string, local *Location) (Time, bool) {
	return parseRFC3339(s, local)
}
