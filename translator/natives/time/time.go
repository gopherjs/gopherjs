// +build js

package time

import (
	"github.com/gopherjs/gopherjs/js"
)

func now() (sec int64, nsec int32) {
	msec := js.Global.Get("Date").New().Call("getTime").Int64()
	return msec / 1000, int32(msec%1000) * 1000000
}

func AfterFunc(d Duration, f func()) *Timer {
	js.Global.Call("setTimeout", f, d/Millisecond)
	return nil
}

func After(d Duration) <-chan Time {
	js.Global.Call("go$notSupported", "time.After (use time.AfterFunc instead)")
	panic("unreachable")
}

func Sleep(d Duration) {
	js.Global.Call("go$notSupported", "time.Sleep (use time.AfterFunc instead)")
	panic("unreachable")
}

func Tick(d Duration) <-chan Time {
	js.Global.Call("go$notSupported", "time.Tick (use time.AfterFunc instead)")
	panic("unreachable")
}

func NewTimer(d Duration) *Timer {
	js.Global.Call("go$notSupported", "time.NewTimer (use time.AfterFunc instead)")
	panic("unreachable")
}
