//go:build js

package sync_test

import "testing"

//gopherjs:replace
func TestIssue40999(t *testing.T) {
	t.Skip("test relies on runtime.SetFinalizer, which GopherJS does not implement")
}
