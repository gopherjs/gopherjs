// +build js

package reflect_test

import (
	"reflect"
	"testing"
)

func TestAlignment(t *testing.T) {
	t.Skip()
}

func TestSliceOverflow(t *testing.T) {
	t.Skip()
}

func TestFuncLayout(t *testing.T) {
	t.Skip()
}

func TestArrayOfDirectIface(t *testing.T) {
	t.Skip()
}

func TestTypelinksSorted(t *testing.T) {
	t.Skip()
}

func TestGCBits(t *testing.T) {
	t.Skip()
}

func TestChanAlloc(t *testing.T) {
	t.Skip()
}

func TestSelectOnInvalid(t *testing.T) {
	reflect.Select([]reflect.SelectCase{
		{
			Dir:  reflect.SelectRecv,
			Chan: reflect.Value{},
		}, {
			Dir:  reflect.SelectSend,
			Chan: reflect.Value{},
			Send: reflect.ValueOf(1),
		}, {
			Dir: reflect.SelectDefault,
		},
	})
}
