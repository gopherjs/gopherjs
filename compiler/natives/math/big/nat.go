// +build js

package big

var cacheBase10 struct {
	nopLocker
	table [64]divisor // cached divisors for base 10
}

type nopLocker struct{}

func (nopLocker) Lock()   {}
func (nopLocker) Unlock() {}
