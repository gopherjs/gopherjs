// This is a test of several different kinds of generics.
// It is purposefully overly complecated for testing purposes.
// This integration test is similar to `compiler.Test_CrossPackageAnalysis`.

package main

import (
	"fmt"

	"github.com/gopherjs/gopherjs/tests/testdata/gencircle/trammel/cmp"
	"github.com/gopherjs/gopherjs/tests/testdata/gencircle/trammel/collections"
	"github.com/gopherjs/gopherjs/tests/testdata/gencircle/trammel/stable"
)

type StableMap[K cmp.Ordered, V any] map[K]V

func (m StableMap[K, V]) String() string {
	return stable.MapString(m, func(k K, v V) string {
		return fmt.Sprintf(`%v: %v`, k, v)
	})
}

type SIMap = StableMap[string, int]
type ISMap = StableMap[int, string]

func main() {
	m1 := SIMap{}
	collections.Populate(m1,
		[]string{"one", "two", "three"},
		[]int{1, 2, 3})
	println(m1.String())

	m2 := ISMap{}
	collections.Populate(m2,
		[]int{4, 5, 6},
		[]string{"four", "five", "six"})
	println(m2.String())
}
