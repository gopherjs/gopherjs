// +build js

package atomic_test

import "testing"

func TestHammerStoreLoad(t *testing.T) {
	t.Skip("use of unsafe")
}

func shouldPanic(t *testing.T, name string, f func()) {
	defer func() {
		if recover() == nil {
			t.Errorf("%s did not panic", name)
		}
	}()
	f()
}
