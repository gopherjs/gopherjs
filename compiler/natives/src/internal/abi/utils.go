//go:build js

package abi

// GOPHERJS: These utils are being added because they are common between
// reflect and reflectlite.

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

//gopherjs:new Added to avoid a dependency on strconv.Unquote
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

//gopherjs:new
func GetJsTag(tag string) string {
	for tag != "" {
		// skip leading space
		i := 0
		for i < len(tag) && tag[i] == ' ' {
			i++
		}
		tag = tag[i:]
		if tag == "" {
			break
		}

		// scan to colon.
		// a space or a quote is a syntax error
		i = 0
		for i < len(tag) && tag[i] != ' ' && tag[i] != ':' && tag[i] != '"' {
			i++
		}
		if i+1 >= len(tag) || tag[i] != ':' || tag[i+1] != '"' {
			break
		}
		name := string(tag[:i])
		tag = tag[i+1:]

		// scan quoted string to find value
		i = 1
		for i < len(tag) && tag[i] != '"' {
			if tag[i] == '\\' {
				i++
			}
			i++
		}
		if i >= len(tag) {
			break
		}
		qvalue := string(tag[:i+1])
		tag = tag[i+1:]

		if name == "js" {
			value, _ := unquote(qvalue)
			return value
		}
	}
	return ""
}
