// +build js

package sync

import (
	"unsafe"
)

type Pool struct {
	local     unsafe.Pointer
	localSize uintptr

	store []interface{}
	New   func() interface{}
}

func (p *Pool) Get() interface{} {
	if len(p.store) == 0 {
		if p.New != nil {
			return p.New()
		}
		return nil
	}
	x := p.store[len(p.store)-1]
	p.store = p.store[:len(p.store)-1]
	return x
}

func (p *Pool) Put(x interface{}) {
	if x == nil {
		return
	}
	p.store = append(p.store, x)
}

func runtime_registerPoolCleanup(cleanup func()) {
}

var semWaiters = make(map[*uint32][]chan struct{})

func runtime_Semacquire(s *uint32) {
	if *s == 0 {
		ch := make(chan struct{})
		semWaiters[s] = append(semWaiters[s], ch)
		<-ch
	}
	*s--
}

func runtime_Semrelease(s *uint32) {
	*s++

	w := semWaiters[s]
	if len(w) == 0 {
		return
	}

	ch := w[0]
	if len(w) == 1 {
		delete(semWaiters, s)
	} else {
		semWaiters[s] = w[1:]
	}

	ch <- struct{}{}
}

func runtime_Syncsemcheck(size uintptr) {
}

func (c *copyChecker) check() {
}
