// +build js

package fmtsort_test

import (
	"math"
	"reflect"
	"testing"

	"internal/fmtsort"
)

// needsSkip reports whether the kind doesn't work for sorting on GopherJS.
func needsSkip(k reflect.Kind) bool {
	switch k {
	case reflect.Ptr, reflect.Chan:
		return true
	}
	return false
}

// Note: sync with the original TestCompare.
func TestCompare(t *testing.T) {
	for _, test := range compareTests {
		for i, v0 := range test {
			for j, v1 := range test {
				// GopherJS specific kind check
				if needsSkip(v0.Kind()) || needsSkip(v1.Kind()) {
					continue
				}

				c := fmtsort.Compare(v0, v1)
				var expect int
				switch {
				case i == j:
					expect = 0
					// NaNs are tricky.
					if typ := v0.Type(); (typ.Kind() == reflect.Float32 || typ.Kind() == reflect.Float64) && math.IsNaN(v0.Float()) {
						expect = -1
					}
				case i < j:
					expect = -1
				case i > j:
					expect = 1
				}
				if c != expect {
					t.Errorf("%s: compare(%v,%v)=%d; expect %d", v0.Type(), v0, v1, c, expect)
				}
			}
		}
	}
}

func TestOrder(t *testing.T) {
	for _, test := range sortTests {
		switch test.data.(type) {
		case map[*int]string, map[chan int]string:
			// GopherJS doesn't support comparison/ordering of such
			// types, unlike "native" architectures
			continue
		case map[[2]int]string:
			// A case of https://github.com/gopherjs/gopherjs/issues/773
			continue
		}
		got := sprint(test.data)
		if got != test.print {
			t.Errorf("%s: got %q, want %q", reflect.TypeOf(test.data), got, test.print)
		}
	}
}
