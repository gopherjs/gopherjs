// +build js

package template

func count(n int) []string {
	if n == 0 {
		return nil
	}
	s := make([]string, n)
	for i := 0; i < n; i++ {
		s[i] = "abcdefghijklmnop"[i : i+1]
	}
	return s
}
