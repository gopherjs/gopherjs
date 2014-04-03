package json

var pool []*encodeState

func newEncodeState() *encodeState {
	if len(pool) == 0 {
		return &encodeState{}
	}
	e := pool[len(pool)-1]
	pool = pool[:len(pool)-1]
	e.Reset()
	return e
}

func putEncodeState(e *encodeState) {
	pool = append(pool, e)
}
