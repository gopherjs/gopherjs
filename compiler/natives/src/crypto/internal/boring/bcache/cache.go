//go:build js

package bcache

import "unsafe"

// Cache relies on GC to periodically clear the cache.
// Since GopherJS doesn't have the same GC hooks, it currently can not
// register this cache with the GC.
// Without this cache Boring crypto, in particular public and private
// RSA and ECDSA keys, will be slower because the cache will always miss.
type Cache struct{}

func (c *Cache) Register()                           {}
func (c *Cache) Clear()                              {}
func (c *Cache) Get(k unsafe.Pointer) unsafe.Pointer { return nil }
func (c *Cache) Put(k, v unsafe.Pointer)             {}

//gopherjs:purge
func (c *Cache) table() *[cacheSize]unsafe.Pointer

//gopherjs:purge
type cacheEntry struct{}

//gopherjs:purge
func registerCache(unsafe.Pointer)

//gopherjs:purge
const cacheSize = 1021
