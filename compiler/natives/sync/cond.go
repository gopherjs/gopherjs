// +build js

package sync

type Cond struct {
	L Locker

	sema    syncSema
	waiters uint32
	checker copyChecker

	n  int
	ch chan bool
}

func NewCond(l Locker) *Cond {
	return &Cond{L: l, ch: make(chan bool)}
}

func (c *Cond) Wait() {
	c.n++
	c.L.Unlock()
	<-c.ch
	c.L.Lock()
}

func (c *Cond) Signal() {
	if c.n == 0 {
		return
	}
	c.n--
	c.ch <- true
}

func (c *Cond) Broadcast() {
	n := c.n
	c.n = 0
	for i := 0; i < n; i++ {
		c.ch <- true
	}
}

func runtime_Syncsemcheck(size uintptr) {
}
