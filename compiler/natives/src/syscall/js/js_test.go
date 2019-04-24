// +build js

package js_test

import (
	"testing"
)

func TestIntConversion(t *testing.T) {
	testIntConversion(t, 0)
	testIntConversion(t, 1)
	testIntConversion(t, -1)
	testIntConversion(t, 1<<20)
	testIntConversion(t, -1<<20)
	// Skip too-big numbers
	// testIntConversion(t, 1<<40)
	// testIntConversion(t, -1<<40)
	// testIntConversion(t, 1<<60)
	// testIntConversion(t, -1<<60)
}
