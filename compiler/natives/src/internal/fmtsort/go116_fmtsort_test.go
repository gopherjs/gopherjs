// +build js
// +build go1.16

package fmtsort_test

import (
	"reflect"
	"testing"
)

func ct(typ reflect.Type, args ...interface{}) []reflect.Value {
	return nil
}

func TestCompare(t *testing.T) {
	t.Skip("known issue: unsafe.Pointer doesn't support")
}

func TestOrder(t *testing.T) {
	t.Skip("known issue: nil key doesn't work")
}
