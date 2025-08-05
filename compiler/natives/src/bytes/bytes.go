//go:build js

package bytes

//gopherjs:replace
func IndexByte(s []byte, c byte) int {
	for i, b := range s {
		if b == c {
			return i
		}
	}
	return -1
}

//gopherjs:replace
func Equal(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i, c := range a {
		if c != b[i] {
			return false
		}
	}
	return true
}

//gopherjs:replace
func Compare(a, b []byte) int {
	for i, ca := range a {
		if i >= len(b) {
			return 1
		}
		cb := b[i]
		if ca < cb {
			return -1
		}
		if ca > cb {
			return 1
		}
	}
	if len(a) < len(b) {
		return -1
	}
	return 0
}
