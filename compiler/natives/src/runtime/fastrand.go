// +build js

package runtime

import (
	"github.com/goplusjs/gopherjs/js"
)

func rand32() uint32 {
	return uint32(js.Global.Get("Math").Call("random").Float() * (1<<32 - 1))
}

var (
	fastrand0 uint32
	fastrand1 uint32
)

func init() {
	fastrand0 = rand32()
	fastrand1 = rand32()
}

func fastrand() uint32 {
	// Implement xorshift64+: 2 32-bit xorshift sequences added together.
	// Shift triplet [17,7,16] was calculated as indicated in Marsaglia's
	// Xorshift paper: https://www.jstatsoft.org/article/view/v008i14/xorshift.pdf
	// This generator passes the SmallCrush suite, part of TestU01 framework:
	// http://simul.iro.umontreal.ca/testu01/tu01.html
	s1, s0 := fastrand0, fastrand1
	s1 ^= s1 << 17
	s1 = s1 ^ s0 ^ s1>>7 ^ s0>>16
	fastrand0, fastrand1 = s0, s1
	return s0 + s1
}
