package tests

import (
	"sort"
	"testing"
)

func TestSortSlice(t *testing.T) {
	a := [...]int{5, 4, 3, 2, 1}
	s := a[1:4]
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	if a != [...]int{5, 2, 3, 4, 1} {
		t.Fatalf("not equal")
	}
}
