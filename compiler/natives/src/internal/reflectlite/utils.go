//go:build js

package reflectlite

//gopherjs:new
type errorString struct {
	s string
}

//gopherjs:new
func (e *errorString) Error() string {
	return e.s
}

//gopherjs:new
var ErrSyntax = &errorString{"invalid syntax"}

//gopherjs:new
func unquote(s string) (string, error) {
	if len(s) < 2 {
		return s, nil
	}
	if s[0] == '\'' || s[0] == '"' {
		if s[len(s)-1] == s[0] {
			return s[1 : len(s)-1], nil
		}
		return "", ErrSyntax
	}
	return s, nil
}
