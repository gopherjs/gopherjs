// +build js
// +build go1.14

package time

func resetTimer(r *runtimeTimer, w int64) bool {
	active := stopTimer(r)
	r.when = w
	startTimer(r)
	return active
}
