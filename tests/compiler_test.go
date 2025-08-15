package tests

import "testing"

func TestVariadicNil(t *testing.T) {
	t.Run("only variadic", func(t *testing.T) {
		printVari := func(strs ...string) []string {
			return strs
		}

		if got := printVari(); got != nil {
			t.Errorf("printVari(): got: %#v; want %#v.", got, nil)
		}

		{
			var want []string
			if got := printVari(want...); got != nil {
				t.Errorf("printVari(want...): got: %#v; want %#v.", got, nil)
			}
		}

		{
			want := []string{}
			if got := printVari(want...); got == nil || len(got) != len(want) {
				t.Errorf("printVari(want...): got: %#v; want %#v.", got, want)
			}
		}
	})
	t.Run("mixed", func(t *testing.T) {
		printVari := func(_ int, strs ...string) []string {
			return strs
		}

		if got := printVari(0); got != nil {
			t.Errorf("printVari(): got: %#v; want %#v.", got, nil)
		}

		{
			var want []string
			if got := printVari(0, want...); got != nil {
				t.Errorf("printVari(want...): got: %#v; want %#v.", got, nil)
			}
		}

		{
			want := []string{}
			if got := printVari(0, want...); got == nil || len(got) != len(want) {
				t.Errorf("printVari(want...): got: %#v; want %#v.", got, want)
			}
		}
	})
}
