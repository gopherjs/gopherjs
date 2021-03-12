// +build js

package testing

func TestTBHelper(t *T) {
	t.Skip("t.Helper() is not supported by GopherJS.")
}

func TestTBHelperParallel(t *T) {
	t.Skip("t.Helper() is not supported by GopherJS.")
}
