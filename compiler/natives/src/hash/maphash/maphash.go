//go:build js

package maphash

import (
	_ "unsafe" // for linkname
)

// hashkey is similar how it is defined in runtime/alg.go for Go 1.19
// to be used in hash{32,64}.go to seed the hash function as part of
// runtime_memhash. We're using locally defined memhash so it got moved here.
//
//gopherjs:new
var hashkey [3]uint32

func init() {
	for i := range hashkey {
		hashkey[i] = runtime_fastrand() | 1
		// The `| 1` is to make sure these numbers are odd
	}
}

//go:linkname runtime_fastrand runtime.fastrand
//gopherjs:new
func runtime_fastrand() uint32

// Bytes uses less efficient equivalent to avoid using unsafe.
//
//gopherjs:replace
func Bytes(seed Seed, b []byte) uint64 {
	var h Hash
	h.SetSeed(seed)
	_, _ = h.Write(b)
	return h.Sum64()
}

// String uses less efficient equivalent to avoid using unsafe.
//
//gopherjs:replace
func String(seed Seed, s string) uint64 {
	var h Hash
	h.SetSeed(seed)
	_, _ = h.WriteString(s)
	return h.Sum64()
}

// rthash is similar to the Go 1.19.13 version
// with the call to memhash changed to not use unsafe pointers.
//
//gopherjs:replace
func rthash(b []byte, seed uint64) uint64 {
	if len(b) == 0 {
		return seed
	}
	// The runtime hasher only works on uintptr. Since GopherJS implements a
	// 32-bit environment, we use two parallel hashers on the lower and upper 32
	// bits.
	lo := memhash(b, uint32(seed))
	hi := memhash(b, uint32(seed>>32))
	return uint64(hi)<<32 | uint64(lo)
}

//gopherjs:purge to remove link using unsafe pointers, use memhash instead.
func runtime_memhash()

// The implementation below is adapted from the upstream runtime/hash32.go
// and avoids use of unsafe, which GopherJS doesn't support well and leads to
// worse performance.
//
// Note that this hashing function is not actually used by GopherJS maps, since
// we use JS maps instead, but it may be still applicable for use with custom
// map types.
//
// Hashing algorithm inspired by wyhash:
// https://github.com/wangyi-fudan/wyhash/blob/ceb019b530e2c1c14d70b79bfa2bc49de7d95bc1/Modern%20Non-Cryptographic%20Hash%20Function%20and%20Pseudorandom%20Number%20Generator.pdf
//
//gopherjs:new
func memhash(p []byte, seed uint32) uintptr {
	s := len(p)
	a, b := mix32(uint32(seed), uint32(s)^hashkey[0])
	if s == 0 {
		return uintptr(a ^ b)
	}
	for ; s > 8; s -= 8 {
		a ^= readUnaligned32(p)
		b ^= readUnaligned32(add(p, 4))
		a, b = mix32(a, b)
		p = add(p, 8)
	}
	if s >= 4 {
		a ^= readUnaligned32(p)
		b ^= readUnaligned32(add(p, s-4))
	} else {
		t := uint32(p[0])
		t |= uint32(add(p, s>>1)[0]) << 8
		t |= uint32(add(p, s-1)[0]) << 16
		b ^= t
	}
	a, b = mix32(a, b)
	a, b = mix32(a, b)
	return uintptr(a ^ b)
}

//gopherjs:new
func add(p []byte, x int) []byte {
	return p[x:]
}

// Note: These routines perform the read in little endian.
//
//gopherjs:new
func readUnaligned32(p []byte) uint32 {
	return uint32(p[0]) | uint32(p[1])<<8 | uint32(p[2])<<16 | uint32(p[3])<<24
}

//gopherjs:new
func mix32(a, b uint32) (uint32, uint32) {
	c := uint64(a^uint32(hashkey[1])) * uint64(b^uint32(hashkey[2]))
	return uint32(c), uint32(c >> 32)
}

/*
	The following functions were modified in Go 1.17 to improve performance,
	but at the expense of being unsafe, and thus incompatible with GopherJS.
	See https://cs.opensource.google/go/go/+/refs/tags/go1.19.13:src/hash/maphash/maphash.go;
	To compensate, we use a simplified version of each method from Go 1.19.13,
	similar to Go 1.16's versions, with the call to rthash changed to not use unsafe pointers.

	See upstream issue https://github.com/golang/go/issues/47342 to implement
	a purego version of this package, which should render this hack (and
	likely this entire file) obsolete.
*/

// Write is a simplification from Go 1.19 changed to not use unsafe.
//
//gopherjs:replace
func (h *Hash) Write(b []byte) (int, error) {
	size := len(b)
	if h.n+len(b) > bufSize {
		h.initSeed()
		for h.n+len(b) > bufSize {
			k := copy(h.buf[h.n:], b)
			h.state.s = rthash(h.buf[:], h.state.s)
			b = b[k:]
			h.n = 0
		}
	}
	h.n += copy(h.buf[h.n:], b)
	return size, nil
}

// WriteString is a simplification from Go 1.19 changed to not use unsafe.
//
//gopherjs:replace
func (h *Hash) WriteString(s string) (int, error) {
	size := len(s)
	if h.n+len(s) > bufSize {
		h.initSeed()
		for h.n+len(s) > bufSize {
			k := copy(h.buf[h.n:], s)
			h.state.s = rthash(h.buf[:], h.state.s)
			s = s[k:]
			h.n = 0
		}
	}
	h.n += copy(h.buf[h.n:], s)
	return size, nil
}

// flush is the Go 1.19 version changed to not use unsafe.
//
//gopherjs:replace
func (h *Hash) flush() {
	if h.n != len(h.buf) {
		panic("maphash: flush of partially full buffer")
	}
	h.initSeed()
	h.state.s = rthash(h.buf[:], h.state.s)
	h.n = 0
}

// Sum64 is the Go 1.19 version changed to not use unsafe.
//
//gopherjs:replace
func (h *Hash) Sum64() uint64 {
	h.initSeed()
	return rthash(h.buf[:h.n], h.state.s)
}
