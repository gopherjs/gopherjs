package main

type set map[interface{}]struct{}
type key struct{ a int }

var m = set{}

func deferredDelete(k key) {
	// This built-in deferral will transpile into a "delete" statement wrapped
	// into a proxy lambda. This test ensures we correctly assign proxy lambda
	// argument types.
	defer delete(m, k)
}

func main() {
	k := key{a: 42}
	m[k] = struct{}{}
	deferredDelete(k)
	if _, found := m[k]; found {
		panic("deferred delete didn't work!")
	}
}
