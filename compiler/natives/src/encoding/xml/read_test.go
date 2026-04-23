//go:build js

package xml

import "testing"

func TestCVE202228131(t *testing.T) {
	t.Skip(`Uses maxUnmarshalDepth in a depth test that will exceed maximum call stack in JS`)
}
