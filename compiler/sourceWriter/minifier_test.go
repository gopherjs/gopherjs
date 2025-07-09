package sourceWriter

import (
	"bytes"
	"math/rand"
	"testing"
)

func TestMinifier(t *testing.T) {
	tests := []struct {
		input       string
		exp         string
		skipOrig    bool
		doNotMinify bool
	}{
		{
			input: "",
			exp:   "",
		}, {
			input: " ",
			exp:   "",
		}, {
			input: "\t",
			exp:   "",
		}, {
			input: "\n",
			exp:   "",
		}, {
			input: "   leading",
			exp:   "leading",
		}, {
			skipOrig: true, // original fails with `index out of range`
			input:    "tailing   ",
			exp:      "tailing",
		}, {
			input: "a",
			exp:   "a",
		}, {
			input: "a b",
			exp:   "a b",
		}, {
			input: "a  b",
			exp:   "a b",
		}, {
			input: "a\tb",
			exp:   "a\tb",
		}, {
			input: "a\nb",
			exp:   "a\nb",
		}, {
			input: "a\n\t b",
			exp:   "a b",
		}, {
			input: "a \nb",
			exp:   "a\nb",
		}, {
			input: "-  -",
			exp:   "- -",
		}, {
			input: "( ( ) )",
			exp:   "(())",
		}, {
			// doesn't seem right but matches original behavior
			input: "/ *",
			exp:   "/*",
		}, {
			// doesn't seem right but matches original behavior
			input: "hello/*small blue*/world",
			exp:   "helloworld",
		}, {
			skipOrig: true, // original didn't handle single line comments
			input:    "hello//small blue\nworld",
			exp:      "hello\nworld",
		}, {
			skipOrig: true, // original returns "hellonfinished comment"
			input:    "hello/*unfinished comment",
			exp:      "hello",
		}, {
			skipOrig: true, // original didn't handle single line comments
			input:    "hello//eof comment",
			exp:      "hello",
		}, {
			skipOrig: true, // original fails with `index out of range`
			input:    "ending with comment start/",
			exp:      "ending with comment start/",
		}, {
			input: `string   "hello\n   \"world\""  literal`,
			exp:   `string"hello\n   \"world\""literal`,
		}, {
			skipOrig: true, // original fails with `index out of range`
			input:    `hello  "unfinished string literal`,
			exp:      `hello"unfinished string literal`,
		}, {
			skipOrig: true, // original fails with `index out of range`
			input:    `ending with string literal start "`,
			exp:      `ending with string literal start"`,
		},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			var buf bytes.Buffer
			min := newMinifier(&buf, !test.doNotMinify)
			_, err := min.Write([]byte(test.input))
			if err != nil {
				t.Fatalf("Write failed: %v", err)
			}
			if got := buf.String(); got != test.exp {
				t.Errorf("expected %q, got %q", test.exp, got)
			}

			if !test.skipOrig {
				orig := string(originalRemoveWhitespace([]byte(test.input), true))
				if orig != test.exp {
					t.Errorf("expected %q, orig %q", test.exp, orig)
				}
			}
		})
	}
}

func BenchmarkMinifier(b *testing.B) {
	r := rand.New(rand.NewSource(0))

	inputCount := 1000
	inputs := make([][]byte, inputCount)
	for i := 0; i < inputCount; i++ {
		inputs[i] = generateRandomInput(r)

		// Confirm they both get the same result on the random input.
		buf := &bytes.Buffer{}
		_, _ = newMinifier(buf, true).Write(inputs[i])
		got := buf.Bytes()
		orig := originalRemoveWhitespace(inputs[i], true)
		if !bytes.Equal(got, orig) {
			b.Errorf("minified input %d does not match original:\n\tinput: %q\n\tgot %q\n\twant %q", i, inputs[i], got, orig)
		}
	}

	b.Run("minify", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			buf := &bytes.Buffer{}
			_, _ = newMinifier(buf, true).Write(inputs[i%inputCount])
		}
	})

	b.Run("original", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = originalRemoveWhitespace(inputs[i%inputCount], true)
		}
	})
}

func originalRemoveWhitespace(b []byte, minify bool) []byte {
	if !minify {
		return b
	}

	var out []byte
	var previous byte
	for len(b) > 0 {
		switch b[0] {
		case '\b':
			out = append(out, b[:5]...)
			b = b[5:]
			continue
		case ' ', '\t', '\n':
			if (!needsSpace(previous) || !needsSpace(b[1])) && !(previous == '-' && b[1] == '-') {
				b = b[1:]
				continue
			}
		case '"':
			out = append(out, '"')
			b = b[1:]
			for {
				i := bytes.IndexAny(b, "\"\\")
				out = append(out, b[:i]...)
				b = b[i:]
				if b[0] == '"' {
					break
				}
				// backslash
				out = append(out, b[:2]...)
				b = b[2:]
			}
		case '/':
			if b[1] == '*' {
				i := bytes.Index(b[2:], []byte("*/"))
				b = b[i+4:]
				continue
			}
		}
		out = append(out, b[0])
		previous = b[0]
		b = b[1:]
	}
	return out
}

func generateRandomInput(r *rand.Rand) []byte {
	b := &bytes.Buffer{}
	prevPick := -1
	partCount := r.Intn(1000) + 1000
	gens := []func(r *rand.Rand) []byte{
		generateRandomWhitespace,
		generateRandomSymbol,
		generateRandomString,
		generateRandomWord,
		generateRandomComment,
	}

	for i := 2; i < partCount; i++ {
		pick := r.Intn(len(gens) - 1)
		if pick == prevPick {
			pick++ // ensure we don't repeat the same type in a row
		}
		_, _ = b.Write(gens[pick](r))
		prevPick = pick
	}
	// since the original has trouble if the last two parts are
	// non-word followed by space, just always add a word
	_, _ = b.Write(generateRandomWord(r))
	return b.Bytes()
}

func generateRandomWord(r *rand.Rand) []byte {
	const wordParts = `abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_$`
	return generateRandomChunk(r, wordParts, r.Intn(30)+1)
}

func generateRandomSymbol(r *rand.Rand) []byte {
	const symbols = `+-*/%&|^!=<>:;,.()[]{}?@~`
	sym := generateRandomChunk(r, symbols, r.Intn(4)+1)
	// avoid generating a comment start
	sym = bytes.ReplaceAll(sym, []byte(`/`), []byte(`/ `))
	return sym
}

func generateRandomString(r *rand.Rand) []byte {
	const stringParts = `abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 "\`
	str := generateRandomChunk(r, stringParts, r.Intn(50)+5)
	str = bytes.ReplaceAll(str, []byte(`\`), []byte(`\\`))
	str = bytes.ReplaceAll(str, []byte(`"`), []byte(`\"`))
	str = append(append([]byte{'"'}, str...), '"')
	return str
}
func generateRandomComment(r *rand.Rand) []byte {
	const stringParts = `abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 "\-+&`
	str := generateRandomChunk(r, stringParts, r.Intn(50)+5)
	str = append(append([]byte(`/*`), str...), '*', '/')
	return str
}

func generateRandomWhitespace(r *rand.Rand) []byte {
	const whitespace = " \t\n"
	return generateRandomChunk(r, whitespace, r.Intn(3)+1)
}

func generateRandomChunk(r *rand.Rand, source string, length int) []byte {
	word := make([]byte, length)
	for i := 0; i < length; i++ {
		word[i] = source[r.Intn(len(source))]
	}
	return word
}
