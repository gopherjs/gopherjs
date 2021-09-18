//go:build js
// +build js

package field

import (
	"testing/quick"
)

var quickCheckConfig1024 = &quick.Config{MaxCount: 100}
