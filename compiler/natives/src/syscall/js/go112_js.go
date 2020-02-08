// +build js
// +build !go1.13

package js

func (v Value) String() string {
	return v.internal().String()
}
