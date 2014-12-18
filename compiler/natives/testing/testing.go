// +build js

package testing

import "runtime"

type InternalTest struct {
	Name string
	F2   func(*T)
}

func (test *InternalTest) F(t *T) {
	defer func() {
		err := recover()
		if e, ok := err.(*runtime.NotSupportedError); ok {
			t.Skip(e)
		}
		if err != nil {
			panic(err)
		}
	}()
	test.F2(t) //gopherjs:blocking
}
