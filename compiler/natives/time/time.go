// +build js

package time

import (
	"strings"

	"github.com/gopherjs/gopherjs/js"
)

type runtimeTimer struct {
	i       int32
	when    int64
	period  int64
	f       func(interface{}, uintptr)
	arg     interface{}
	timeout *js.Object
	active  bool
}

func initLocal() {
	d := js.Global.Get("Date").New()
	s := d.String()
	i := strings.IndexByte(s, '(')
	j := strings.IndexByte(s, ')')
	if i == -1 || j == -1 {
		localLoc.name = "UTC"
		return
	}
	localLoc.name = s[i+1 : j]
	localLoc.zone = []zone{{localLoc.name, d.Call("getTimezoneOffset").Int() * -60, false}}
}

func runtimeNano() int64 {
	return js.Global.Get("Date").New().Call("getTime").Int64() * int64(Millisecond)
}

func now() (sec int64, nsec int32) {
	n := runtimeNano()
	return n / int64(Second), int32(n % int64(Second))
}

func Sleep(d Duration) {
	c := make(chan struct{})
	js.Global.Call("setTimeout", func() { close(c) }, int(d/Millisecond))
	<-c
}

func startTimer(t *runtimeTimer) {
	t.active = true
	diff := (t.when - runtimeNano()) / int64(Millisecond)
	if diff > 1<<31-1 { // math.MaxInt32
		return
	}
	if diff < 0 {
		diff = 0
	}
	t.timeout = js.Global.Call("setTimeout", func() {
		t.active = false
		go t.f(t.arg, 0)
		if t.period != 0 {
			t.when += t.period
			startTimer(t)
		}
	}, diff+1)
}

func stopTimer(t *runtimeTimer) bool {
	js.Global.Call("clearTimeout", t.timeout)
	wasActive := t.active
	t.active = false
	return wasActive
}
