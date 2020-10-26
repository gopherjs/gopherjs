// +build js
// +build go1.15

package time

import "github.com/gopherjs/gopherjs/js"

type runtimeTimer struct {
	i       int32
	when    int64
	period  int64
	f       func(interface{}, uintptr)
	arg     interface{}
	timeout *js.Object
	active  bool
	seq     uintptr
}

func (t *Ticker) Reset(d Duration) {
	if t.r.f == nil {
		panic("time: Reset called on uninitialized Ticker")
	}
	stopTimer(&t.r)
	c := make(chan Time, 1)
	t.C = c
	t.r = runtimeTimer{
		when:   when(d),
		period: int64(d),
		f:      sendTime,
		arg:    c,
	}
	startTimer(&t.r)
}
