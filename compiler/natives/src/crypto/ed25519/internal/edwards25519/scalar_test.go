//go:build js
// +build js

package edwards25519

import (
	"testing/quick"
)

var quickCheckConfig32 = &quick.Config{MaxCount: 100}
