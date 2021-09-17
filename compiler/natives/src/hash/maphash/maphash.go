//go:build js
// +build js

package maphash

// used in hash{32,64}.go to seed the hash function
var hashkey [4]uint32

func init() {
	for i := range hashkey {
		hashkey[i] = runtime_fastrand()
	}
	hashkey[0] |= 1 // make sure these numbers are odd
	hashkey[1] |= 1
	hashkey[2] |= 1
	hashkey[3] |= 1
}

func _rthash(b []byte, seed uint64) uint64 {
	if len(b) == 0 {
		return seed
	}
	// The runtime hasher only works on uintptr. Since GopherJS implements a
	// 32-bit environment, we use two parallel hashers on the lower and upper 32
	// bits.
	lo := memhash(b, uint32(seed), uint32(len(b)))
	hi := memhash(b, uint32(seed>>32), uint32(len(b)))
	return uint64(hi)<<32 | uint64(lo)
}

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
func memhash(p []byte, seed uint32, s uint32) uintptr {
	a, b := mix32(uint32(seed), uint32(s^hashkey[0]))
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

func add(p []byte, x uint32) []byte {
	return p[x:]
}

// Note: These routines perform the read in little endian.
func readUnaligned32(p []byte) uint32 {
	return uint32(p[0]) | uint32(p[1])<<8 | uint32(p[2])<<16 | uint32(p[3])<<24
}

func mix32(a, b uint32) (uint32, uint32) {
	c := uint64(a^uint32(hashkey[1])) * uint64(b^uint32(hashkey[2]))
	return uint32(c), uint32(c >> 32)
}

/*
	The following functions were modified in Go 1.17 to improve performance,
	but at the expense of being unsafe, and thus incompatible with GopherJS.
	To compensate, we have reverted these to the unoptimized Go 1.16 versions
	for now.

	See upstream issue https://github.com/golang/go/issues/47342 to implement
	a purego version of this package, which should render this hack (and
	likely this entire file) obsolete.
*/

// Write is borrowed from Go 1.16.
func (h *Hash) Write(b []byte) (int, error) {
	size := len(b)
	for h.n+len(b) > len(h.buf) {
		k := copy(h.buf[h.n:], b)
		h.n = len(h.buf)
		b = b[k:]
		h.flush()
	}
	h.n += copy(h.buf[h.n:], b)
	return size, nil
}

// WriteString is borrowed from Go 1.16.
func (h *Hash) WriteString(s string) (int, error) {
	size := len(s)
	for h.n+len(s) > len(h.buf) {
		k := copy(h.buf[h.n:], s)
		h.n = len(h.buf)
		s = s[k:]
		h.flush()
	}
	h.n += copy(h.buf[h.n:], s)
	return size, nil
}

func (h *Hash) flush() {
	if h.n != len(h.buf) {
		panic("maphash: flush of partially full buffer")
	}
	h.initSeed()
	h.state.s = _rthash(h.buf[:], h.state.s)
	h.n = 0
}

// Sum64 is borrowed from Go 1.16.
func (h *Hash) Sum64() uint64 {
	h.initSeed()
	return _rthash(h.buf[:h.n], h.state.s)
}
