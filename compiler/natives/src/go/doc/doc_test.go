//go:build js

package doc

import (
	"fmt"
	"testing"
)

func compareSlices(t *testing.T, name string, got, want interface{}, compareElem interface{}) {
	// TODO(nevkontakte): Remove this override after generics are supported.
	// https://github.com/gopherjs/gopherjs/issues/1013.
	switch got.(type) {
	case []*Func:
		got := got.([]*Func)
		want := want.([]*Func)
		compareElem := compareElem.(func(t *testing.T, msg string, got, want *Func))
		if len(got) != len(want) {
			t.Errorf("%s: got %d, want %d", name, len(got), len(want))
		}
		for i := 0; i < len(got) && i < len(want); i++ {
			compareElem(t, fmt.Sprintf("%s[%d]", name, i), got[i], want[i])
		}
	case []*Type:
		got := got.([]*Type)
		want := want.([]*Type)
		compareElem := compareElem.(func(t *testing.T, msg string, got, want *Type))
		if len(got) != len(want) {
			t.Errorf("%s: got %d, want %d", name, len(got), len(want))
		}
		for i := 0; i < len(got) && i < len(want); i++ {
			compareElem(t, fmt.Sprintf("%s[%d]", name, i), got[i], want[i])
		}
	default:
		t.Errorf("unexpected argument type %T", got)
	}
}
