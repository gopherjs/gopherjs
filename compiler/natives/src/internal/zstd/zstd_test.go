//go:build js

package zstd

import "testing"

//gopehrjs:replace
func TestAlloc(t *testing.T) {
	t.Skip(`testing.AllocsPerRun not supported in GopherJS`)
}

//gopehrjs:replace
func bigData(t testing.TB) []byte {
	t.Skip(`test requires access to ../../testdata that is not avaiable in GopherJS`)
	return nil
}
