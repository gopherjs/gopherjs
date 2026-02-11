//go:build js

package reflectlite_test

import (
	"testing"

	. "internal/reflectlite"
)

// TODO: REMOVE (only added to break down the steps of the test)
func TestTypes(t *testing.T) {
	for i, tt := range typeTests {
		println(`>> i    >>`, i)
		println(`>> tt.i >>`, tt.i)
		println(`>> tt.s >>`, tt.s)

		v := ValueOf(tt.i)
		println(`>> v    >>`, v)

		f := Field(v, 0)
		println(`>> f    >>`, f)

		tx := f.Type()
		println(`>> tx   >>`, tx)

		testReflectType(t, i, tx, tt.s)
	}
}
