//go:build js
// +build js

package sync

// A Pool is a set of temporary objects that may be individually saved and
// retrieved.
//
// GopherJS provides a simpler, naive implementation with no synchronization at
// all. This is still correct for the GopherJS runtime because:
//
//  1. JavaScript is single-threaded, so it is impossible for two threads to be
//     accessing the pool at the same moment in time.
//  2. GopherJS goroutine implementation uses cooperative multi-tasking model,
//     which only allows passing control to other goroutines when the function
//     might block.
//
// TODO(nevkontakte): Consider adding a mutex just to be safe if it doesn't
// create a large performance hit.
//
// Note: there is a special handling in the gopherjs/build package that filters
// out all original Pool implementation in order to avoid awkward unused fields
// referenced by dead code.
type Pool struct {
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

// These are referenced by tests, but are no-ops in GopherJS runtime.
func runtime_procPin() int { return 0 }
func runtime_procUnpin()   {}
