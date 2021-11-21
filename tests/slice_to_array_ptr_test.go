package tests

import "testing"

// https://tip.golang.org/ref/spec#Conversions_from_slice_to_array_pointer
func TestSliceToArrayPointerConversion(t *testing.T) {
	// GopherJS uses TypedArray for numeric types and Array for everything else
	// since those are substantially different types, the tests are repeated
	// for both.
	expectOutOfBoundsPanic := func(t *testing.T) {
		t.Helper()
		if recover() == nil {
			t.Error("out-of-bounds conversion of s should panic")
		}
	}

	t.Run("Numeric", func(t *testing.T) {
		s := make([]byte, 2, 4)
		t.Run("NotNil", func(t *testing.T) {
			s0 := (*[0]byte)(s)
			if s0 == nil {
				t.Error("s0 should not be nil")
			}
		})

		t.Run("ElementPointerEquality", func(t *testing.T) {
			s2 := (*[2]byte)(s)
			if &s2[0] != &s[0] {
				t.Error("&s2[0] should match &s[0]")
			}
			s3 := (*[1]byte)(s[1:])
			if &s3[0] != &s[1] {
				t.Error("&s3[0] should match &s[1]")
			}
		})

		t.Run("SliceToLargerArray", func(t *testing.T) {
			defer expectOutOfBoundsPanic(t)
			s4 := (*[4]byte)(s)
			_ = s4
		})

		t.Run("SharedMemory", func(t *testing.T) {
			s2 := (*[2]byte)(s)
			(*s2)[0] = 'x'
			if s[0] != 'x' {
				t.Errorf("s[0] should be changed")
			}

			s3 := (*[1]byte)(s[1:])
			(*s3)[0] = 'y'
			if s[1] != 'y' {
				t.Errorf("s[1] should be changed")
			}
		})

		var q []byte
		t.Run("NilSlice", func(t *testing.T) {
			q0 := (*[0]byte)(q)
			if q0 != nil {
				t.Error("q0 should be nil")
			}
		})

		t.Run("NilSliceToLargerArray", func(t *testing.T) {
			defer expectOutOfBoundsPanic(t)
			q1 := (*[1]byte)(q)
			_ = q1
		})

		t.Run("ZeroLenSlice", func(t *testing.T) {
			u := make([]byte, 0)
			u0 := (*[0]byte)(u)
			if u0 == nil {
				t.Error("u0 should not be nil")
			}
		})
	})

	t.Run("String", func(t *testing.T) {
		s := make([]string, 2, 2)
		t.Run("NotNil", func(t *testing.T) {
			s0 := (*[0]string)(s)
			if s0 == nil {
				t.Error("s0 should not be nil")
			}
		})

		t.Run("ElementPointerEquality", func(t *testing.T) {
			s2 := (*[2]string)(s)
			if &s2[0] != &s[0] {
				t.Error("&s2[0] should match &s[0]")
			}

			t.Skip("non-numeric slice to underlying array conversion is not supported for subslices")
			s3 := (*[1]string)(s[1:])
			if &s3[0] != &s[1] {
				t.Error("&s3[0] should match &s[1]")
			}
		})

		t.Run("SliceToLargerArray", func(t *testing.T) {
			defer expectOutOfBoundsPanic(t)
			s4 := (*[4]string)(s)
			_ = s4
		})

		t.Run("SharedMemory", func(t *testing.T) {
			s2 := (*[2]string)(s)
			(*s2)[0] = "x"
			if s[0] != "x" {
				t.Errorf("s[0] should be changed")
			}

			t.Skip("non-numeric slice to underlying array conversion is not supported for subslices")
			s3 := (*[1]string)(s[1:])
			(*s3)[0] = "y"
			if s[1] != "y" {
				t.Errorf("s[1] should be changed")
			}
		})

		var q []string
		t.Run("NilSlice", func(t *testing.T) {
			q0 := (*[0]string)(q)
			if q0 != nil {
				t.Error("q0 should be nil")
			}
		})

		t.Run("NilSliceToLargerArray", func(t *testing.T) {
			defer expectOutOfBoundsPanic(t)
			q1 := (*[1]string)(q)
			_ = q1
		})

		t.Run("ZeroLenSlice", func(t *testing.T) {
			u := make([]string, 0)
			u0 := (*[0]string)(u)
			if u0 == nil {
				t.Error("u0 should not be nil")
			}
		})
	})
}
