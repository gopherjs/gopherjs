//go:build js

package edwards25519

import "testing/quick"

// Tests in this package use 64-bit math, which is slow under GopherJS. To keep
// test run time reasonable, we reduce the number of test iterations.
var quickCheckConfig32 = &quick.Config{MaxCountScale: 0.5}
