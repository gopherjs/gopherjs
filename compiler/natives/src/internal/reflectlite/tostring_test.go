//go:build js

package reflectlite_test

import "reflect"

// TODO: REMOVE
//
//gopherjs:keep-original
func valueToStringImpl(val reflect.Value) string {
	return _gopherjs_original_valueToStringImpl(val)
}
