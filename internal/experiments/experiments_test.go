package experiments

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseFlags(t *testing.T) {
	type testFlags struct {
		Exp1     bool `flag:"exp1"`
		Exp2     bool `flag:"exp2"`
		Untagged bool
	}

	tests := []struct {
		descr   string
		raw     string
		want    testFlags
		wantErr error
	}{{
		descr: "default values",
		raw:   "",
		want: testFlags{
			Exp1: false,
			Exp2: false,
		},
	}, {
		descr: "true flag",
		raw:   "exp1=true",
		want: testFlags{
			Exp1: true,
			Exp2: false,
		},
	}, {
		descr: "false flag",
		raw:   "exp1=false",
		want: testFlags{
			Exp1: false,
			Exp2: false,
		},
	}, {
		descr: "implicit value",
		raw:   "exp1",
		want: testFlags{
			Exp1: true,
			Exp2: false,
		},
	}, {
		descr: "multiple flags",
		raw:   "exp1=true,exp2=true",
		want: testFlags{
			Exp1: true,
			Exp2: true,
		},
	}, {
		descr: "repeated flag",
		raw:   "exp1=false,exp1=true",
		want: testFlags{
			Exp1: true,
			Exp2: false,
		},
	}, {
		descr: "spaces",
		raw:   " exp1 = true, exp2=true ",
		want: testFlags{
			Exp1: true,
			Exp2: true,
		},
	}, {
		descr: "unknown flags",
		raw:   "Exp1=true,Untagged,Foo=true",
		want: testFlags{
			Exp1:     false,
			Exp2:     false,
			Untagged: false,
		},
	}, {
		descr:   "empty flag name",
		raw:     "=true",
		wantErr: ErrInvalidFormat,
	}, {
		descr:   "invalid flag value",
		raw:     "exp1=foo",
		wantErr: ErrInvalidFormat,
	}}

	for _, test := range tests {
		t.Run(test.descr, func(t *testing.T) {
			got := testFlags{}
			err := parseFlags(test.raw, &got)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Errorf("Got: parseFlags(%q) returned error: %v. Want: %v.", test.raw, err, test.wantErr)
				}
			} else {
				if err != nil {
					t.Fatalf("Got: parseFlags(%q) returned error: %v. Want: no error.", test.raw, err)
				}
				if diff := cmp.Diff(test.want, got); diff != "" {
					t.Fatalf("parseFlags(%q) returned diff (-want,+got):\n%s", test.raw, diff)
				}
			}
		})
	}

	t.Run("invalid dest type", func(t *testing.T) {
		var dest string
		err := parseFlags("", &dest)
		if !errors.Is(err, ErrInvalidDest) {
			t.Fatalf("Got: parseFlags() returned error: %v. Want: %v.", err, ErrInvalidDest)
		}
	})

	t.Run("nil dest", func(t *testing.T) {
		err := parseFlags("", (*struct{})(nil))
		if !errors.Is(err, ErrInvalidDest) {
			t.Fatalf("Got: parseFlags() returned error: %v. Want: %v.", err, ErrInvalidDest)
		}
	})

	t.Run("unsupported flag type", func(t *testing.T) {
		var dest struct {
			Foo string `flag:"foo"`
		}
		err := parseFlags("foo", &dest)
		if !errors.Is(err, ErrInvalidDest) {
			t.Fatalf("Got: parseFlags() returned error: %v. Want: %v.", err, ErrInvalidDest)
		}
	})
}
