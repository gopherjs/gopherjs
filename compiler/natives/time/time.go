// +build js

package time

import (
	"github.com/gopherjs/gopherjs/js"
)

func now() (sec int64, nsec int32) {
	msec := js.Global.Get("Date").New().Call("getTime").Int64()
	return msec / 1000, int32(msec%1000) * 1000000
}

func runtimeNano() int64 {
	msec := js.Global.Get("Date").New().Call("getTime").Int64()
	return msec * 1000000
}

func Sleep(d Duration) {
	c := make(chan struct{})
	js.Global.Call("setTimeout", func() { close(c) }, int(d/Millisecond))
	<-c
}

func startTimer(t *runtimeTimer) {
	diff := int((t.when - runtimeNano()) / 1000000)
	js.Global.Call("setTimeout", func() {
		t.f(runtimeNano(), t.arg)
		if t.period != 0 {
			t.when += t.period
			startTimer(t)
		}
	}, diff)
}

func stopTimer(t *runtimeTimer) bool {
	return false
}
