// +build js

package reflect

type nopRWLocker struct{}

func (nopRWLocker) Lock()    {}
func (nopRWLocker) Unlock()  {}
func (nopRWLocker) RLock()   {}
func (nopRWLocker) RUnlock() {}

// ptrMap is the cache for PtrTo.
var ptrMap struct {
	nopRWLocker
	m map[*rtype]*ptrType
}

// The lookupCache caches ChanOf, MapOf, and SliceOf lookups.
var lookupCache struct {
	nopRWLocker
	m map[cacheKey]*rtype
}

var layoutCache struct {
	nopRWLocker
	m map[layoutKey]layoutType
}
