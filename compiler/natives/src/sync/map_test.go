//go:build js
// +build js

package sync_test

import "testing"

func TestIssue40999(t *testing.T) {
	t.Skip("test relies on runtime.SetFinalizer, which GopherJS does not implement")
}
