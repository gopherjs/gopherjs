package tests

import "testing"

type mySlice []int

func (s mySlice) Len() int { return len(s) }
func TestMethodValueOnNilSlice(t *testing.T) {
	var s mySlice
	f := s.Len
	if got := f(); got != 0 {
		t.Errorf("got %d, want 0", got)
	}
}
