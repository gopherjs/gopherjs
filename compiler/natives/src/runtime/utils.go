// +build js
// +build !go1.15

package runtime

func efaceOf(ep *interface{}) *eface {
	panic("efaceOf: not supported")
}

func throw(s string) {
	panic(errorString(s))
}
