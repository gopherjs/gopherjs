// +build js

package ioutil

var pool [][]byte

func blackHole() []byte {
	if len(pool) == 0 {
		return make([]byte, 8192)
	}
	b := pool[len(pool)-1]
	pool = pool[:len(pool)-1]
	return b
}

func blackHolePut(b []byte) {
	pool = append(pool, b)
}
