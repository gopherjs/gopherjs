package sourcemapx

import (
	"bytes"
	"fmt"
	"go/token"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var encodedHint = []byte{
	104, 101, 108, 108, 111, 44, 32, 119, 111, 114, 108, 100, 33, // "hello, world!"
	HintMagic,  // Magic.
	0x00, 0x04, // Size.
	0x01, 0x02, 0x03, 0x04, // Payload.
	103, 111, 112, 104, 101, 114, 115, // "gophers"
}

func TestFindHint(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		got := FindHint(encodedHint)
		want := 13
		if got != want {
			t.Errorf("Got: FindHint(encodedHint) = %d. Want: %d.", got, want)
		}
		if magic := encodedHint[got]; magic != HintMagic {
			t.Errorf("Got: magic at hint position: %x. Want: %x.", magic, HintMagic)
		}
	})

	t.Run("not found", func(t *testing.T) {
		got := FindHint(encodedHint[14:]) // Slice past the hint location.
		want := -1
		if got != want {
			t.Errorf("Got: FindHint(encodedHint) = %d. Want: %d.", got, want)
		}
	})
}

func TestReadHint(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		tests := []struct {
			descr      string
			bytes      []byte
			wantHint   Hint
			wantLength int
		}{{
			descr:      "1234",
			bytes:      encodedHint[13:],
			wantHint:   Hint{Payload: []byte{1, 2, 3, 4}},
			wantLength: 7,
		}, {
			descr:      "empty",
			bytes:      []byte{HintMagic, 0x00, 0x00},
			wantHint:   Hint{Payload: []byte{}},
			wantLength: 3,
		}}

		for _, test := range tests {
			t.Run(test.descr, func(t *testing.T) {
				// Copy input data to avoid clobbering it further in the test.
				b := make([]byte, len(test.bytes))
				copy(b, test.bytes)

				hint, length := ReadHint(b)
				if length != test.wantLength {
					t.Errorf("Got: read hint length = %d. Want: %d.", length, test.wantLength)
				}
				// Zero out input bytes to make sure returned hint is not backed by the
				// same memory.
				for i := range b {
					b[i] = 0
				}
				if diff := cmp.Diff(test.wantHint, hint); diff != "" {
					t.Errorf("ReadHint() returned diff (-want,+got):\n%s", diff)
				}
			})
		}
	})

	t.Run("panic", func(t *testing.T) {
		tests := []struct {
			descr string
			bytes []byte
			panic string
		}{{
			descr: "incomplete header",
			bytes: []byte{HintMagic, 0x00},
			panic: "too short to contain hint header",
		}, {
			descr: "incomplete payload",
			bytes: []byte{HintMagic, 0x00, 0x01},
			panic: "too short to contain hint payload",
		}, {
			descr: "wrong magic",
			bytes: []byte{'a', 0x00, 0x01, 0x00},
			panic: "doesn't start with magic",
		}}

		for _, test := range tests {
			t.Run(test.descr, func(t *testing.T) {
				defer func() {
					err := recover()
					if err == nil {
						t.Fatalf("Got: no panic. Expected a panic.")
					}
					if !strings.Contains(fmt.Sprint(err), test.panic) {
						t.Errorf("Got panic: %v. Expected to contain: %s.", err, test.panic)
					}
				}()

				ReadHint(test.bytes)
			})
		}
	})
}

func TestHintWrite(t *testing.T) {
	tests := []struct {
		descr string
		hint  Hint
		want  []byte
	}{{
		descr: "empty",
		hint:  Hint{},
		want:  []byte{HintMagic, 0x00, 0x00},
	}, {
		descr: "1234",
		hint:  Hint{Payload: []byte{1, 2, 3, 4}},
		want:  []byte{HintMagic, 0x00, 0x04, 1, 2, 3, 4},
	}}

	for _, test := range tests {
		t.Run(test.descr, func(t *testing.T) {
			buf := &bytes.Buffer{}
			if _, err := test.hint.WriteTo(buf); err != nil {
				t.Fatalf("Got: hint.Write() returned error: %s. Want: no error.", err)
			}
			if diff := cmp.Diff(test.want, buf.Bytes()); diff != "" {
				t.Fatalf("%#v.Write() returned diff (-want,+got):\n%s", test.hint, diff)
			}
		})
	}
}

func TestHintPack(t *testing.T) {
	tests := []struct {
		descr string
		value any
	}{{
		descr: "empty position",
		value: token.NoPos,
	}, {
		descr: "non empty position",
		value: token.Pos(42),
	}, {
		descr: "identifier",
		value: Identifier{
			Name:         "foo$1",
			OriginalName: "foo",
			OriginalPos:  token.Pos(42),
		},
	}}

	for _, test := range tests {
		t.Run(test.descr, func(t *testing.T) {
			h := Hint{}
			if err := h.Pack(test.value); err != nil {
				t.Fatalf("h.Pack(%#v) returned error: %s. Want: no error.", test.value, err)
			}
			unpacked, err := h.Unpack()
			if err != nil {
				t.Fatalf("h.Unpack() returned error: %s. Want: no error.", err)
			}
			if diff := cmp.Diff(test.value, unpacked); diff != "" {
				t.Errorf("Unpacked value doesn't match the original (-want,+got):\n%s", diff)
			}
		})
	}
}
