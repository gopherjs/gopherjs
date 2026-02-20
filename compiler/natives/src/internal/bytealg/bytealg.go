//go:build js

package bytealg

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
func IndexByte(b []byte, c byte) int {
	for i, x := range b {
		if x == c {
			return i
		}
	}
	return -1
}

//gopherjs:replace
func IndexByteString(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

//gopherjs:replace
func MakeNoZero(n int) []byte {
	return make([]byte, n)
}
