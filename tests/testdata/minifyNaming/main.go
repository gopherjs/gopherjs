// This package tests an issue found in slices/zsortanyfunc.go:breakPatternsCmpFunc.
// The issue was that a type pointer and a function variable both get named the
// same name when the code is minified. This code is designed specifically to
// try to hit a package level minified name for a type pointer being the same
// as used in a function for a function level variable.
package main

type number uint64

func (num *number) Inc() uint64 {
	*num++
	return uint64(*num)
}

func account() int {
	num := number(8)
	ninc := int(num.Inc())
	return ninc
}

func main() {
	println("num:", account())
}
