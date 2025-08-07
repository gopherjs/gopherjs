//go:build js

package field

import "testing/quick"

// Tests in this package use 64-bit math, which is slow under GopherJS. To keep
// test run time reasonable, we reduce the number of test iterations.
//
//gopherjs:replace
var quickCheckConfig1024 = &quick.Config{MaxCountScale: 10}
