package tests

import "testing"

func TestSliceToArrayPointerConversion(t *testing.T) {
	// https://tip.golang.org/ref/spec#Conversions_from_slice_to_array_pointer
	s := make([]byte, 2, 4)
	s0 := (*[0]byte)(s)
	if s0 == nil {
		t.Error("s0 should not be nil")
	}
	s2 := (*[2]byte)(s)
	if &s2[0] != &s[0] {
		t.Error("&s2[0] should match &s[0]")
	}
	r := func() (r interface{}) {
		defer func() {
			r = recover()
		}()
		s4 := (*[4]byte)(s)
		_ = s4
		return nil
	}()
	if r == nil {
		t.Error("out-of-bounds conversion of s should panic")
	}

	(*s2)[0] = 'x'
	if s[0] != 'x' {
		t.Errorf("s[0] should be changed")
	}

	var q []string
	q0 := (*[0]string)(q)
	if q0 != nil {
		t.Error("t0 should be nil")
	}
	r = func() (r interface{}) {
		defer func() {
			r = recover()
		}()
		q1 := (*[1]string)(q)
		_ = q1
		return nil
	}
	if r == nil {
		t.Error("out-of-bounds conversion of q should panic")
	}

	u := make([]byte, 0)
	u0 := (*[0]byte)(u)
	if u0 == nil {
		t.Error("u0 should not be nil")
	}
}
