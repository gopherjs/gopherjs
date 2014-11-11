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

// var sems []chan struct{} = []chan struct{}{nil}

// func getSem(s *uint32) chan struct{} {
// 	sem := sems[*s]
// 	if sem == nil {
// 		*s = uint32(len(sems))
// 		sems = append(sems, make(chan struct{}))
// 	}
// 	return sem
// }

// func runtime_Semacquire(s *uint32) {
// 	<-getSem(s)
// }

// func runtime_Semrelease(s *uint32) {
// 	getSem(s) <- struct{}{}
// }

func runtime_Syncsemcheck(size uintptr) {
}

func (c *copyChecker) check() {
}
