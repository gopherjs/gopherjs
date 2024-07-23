//go:build js

package quick

var maxCountCap int = 0

// GopherJSInternalMaxCountCap sets an upper bound of iterations quick test may
// perform. THIS IS GOPHERJS-INTERNAL API, DO NOT USE IT OUTSIDE OF THE GOPHERJS
// CODEBASE, IT MAY CHANGE OR DISAPPEAR WITHOUT NOTICE.
//
// This function can be used to limit run time of standard library tests which
// use testing/quick with too many iterations for GopherJS to complete in a
// reasonable amount of time. This is a better compromise than disabling a slow
// test entirely.
//
//     //gopherjs:keep-original
//     func TestFoo(t *testing.T) {
//         t.Cleanup(quick.GopherJSInternalMaxCountCap(100))
//         _gopherjs_original_TestFoo(t)
//     }

func GopherJSInternalMaxCountCap(newCap int) (restore func()) {
	previousCap := maxCountCap
	maxCountCap = newCap
	return func() {
		maxCountCap = previousCap
	}
}

//gopherjs:keep-original
func (c *Config) getMaxCount() (maxCount int) {
	maxCount = c._gopherjs_original_getMaxCount()
	if maxCountCap > 0 && maxCount > maxCountCap {
		maxCount = maxCountCap
	}
	return maxCount
}
