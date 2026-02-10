//go:build js

package sync

import (
	_ "unsafe" // For go:linkname

	"github.com/gopherjs/gopherjs/js"
)

var semWaiters = make(map[*uint32][]chan bool)

// semAwoken tracks the number of waiters awoken by runtime_Semrelease (`ch <- true`)
// that have not yet acquired the semaphore (`<-ch` in runtime_SemacquireMutex).
//
// This prevents a new call to runtime_SemacquireMutex to wrongly acquire the semaphore
// in between (because runtime_Semrelease has already incremented the semaphore while
// all the pending calls to runtime_SemacquireMutex have not yet received from the channel
// and thus decremented the semaphore).
//
// See https://github.com/gopherjs/gopherjs/issues/736.
var semAwoken = make(map[*uint32]uint32)

func runtime_Semacquire(s *uint32) {
	runtime_SemacquireMutex(s, false, 1)
}

// SemacquireMutex is like Semacquire, but for profiling contended Mutexes.
// Mutex profiling is not supported, so just use the same implementation as runtime_Semacquire.
// TODO: Investigate this. If it's possible to implement, consider doing so, otherwise remove this comment.
func runtime_SemacquireMutex(s *uint32, lifo bool, skipframes int) {
	if (*s - semAwoken[s]) == 0 {
		ch := make(chan bool)
		if lifo {
			semWaiters[s] = append([]chan bool{ch}, semWaiters[s]...)
		} else {
			semWaiters[s] = append(semWaiters[s], ch)
		}
		<-ch
		semAwoken[s] -= 1
		if semAwoken[s] == 0 {
			delete(semAwoken, s)
		}
	}
	*s--
}

func runtime_SemacquireRWMutexR(s *uint32, lifo bool, skipframes int) {
	runtime_SemacquireMutex(s, lifo, skipframes)
}

func runtime_SemacquireRWMutex(s *uint32, lifo bool, skipframes int) {
	runtime_SemacquireMutex(s, lifo, skipframes)
}

func runtime_Semrelease(s *uint32, handoff bool, skipframes int) {
	// TODO: Use handoff if needed/possible.
	*s++

	w := semWaiters[s]
	if len(w) == 0 {
		return
	}

	ch := w[0]
	w = w[1:]
	semWaiters[s] = w
	if len(w) == 0 {
		delete(semWaiters, s)
	}

	semAwoken[s] += 1

	ch <- true
}

func runtime_notifyListCheck(size uintptr) {}

func runtime_canSpin(i int) bool {
	return false
}

//go:linkname runtime_nanotime runtime.nanotime
func runtime_nanotime() int64

// Implemented in runtime.
func throw(s string) {
	js.Global.Call("$throwRuntimeError", s)
}

// GOPHERJS: This is identical to the original but without the go:linkname
// that can not be handled right now, "can not insert local implementation..."
// TODO(grantnelson-wf): Remove once linking works both directions.
//
//gopherjs:replace
func syscall_hasWaitingReaders(rw *RWMutex) bool {
	r := rw.readerCount.Load()
	return r < 0 && r+rwmutexMaxReaders > 0
}
