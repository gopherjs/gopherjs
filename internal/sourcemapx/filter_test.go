package sourcemapx

import (
	"bytes"
	"fmt"
	"go/token"
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFilter(t *testing.T) {
	type entry struct {
		GenLine  int
		GenCol   int
		OrigPos  token.Position
		OrigName string
	}
	entries := []entry{}

	code := &bytes.Buffer{}

	filter := &Filter{
		Writer: code,
		MappingCallback: func(generatedLine, generatedColumn int, originalPos token.Position, originalName string) {
			entries = append(entries, entry{
				GenLine:  generatedLine,
				GenCol:   generatedColumn,
				OrigPos:  originalPos,
				OrigName: originalName,
			})
		},
		FileSet: token.NewFileSet(),
	}

	{
		f := filter.FileSet.AddFile("foo.go", filter.FileSet.Base(), 42)
		f.AddLine(0)
		f.AddLine(10)
		f.AddLine(30)
	}
	writeHint(t, filter, token.NoPos)
	fmt.Fprintln(filter, "Hello")
	writeHint(t, filter, token.Pos(16))
	fmt.Fprintln(filter, "World")

	ident := Identifier{
		Name:         "foo$1",
		OriginalName: "main.Foo",
		OriginalPos:  token.Pos(32),
	}
	fmt.Fprintf(filter, "var x = %sfunction %s() {};\n", ident.EncodeHint(), ident)

	wantCode := `Hello
World
var x = function foo$1() {};
`
	if diff := cmp.Diff(wantCode, code.String()); diff != "" {
		t.Errorf("Generated code differs from expected (-want,+got):\n%s", diff)
	}

	wantEntries := []entry{
		{GenLine: 1},
		{GenLine: 2, OrigPos: token.Position{Filename: "foo.go", Line: 2, Column: 6, Offset: 15}},
		{GenLine: 3, GenCol: 8, OrigPos: token.Position{Filename: "foo.go", Line: 3, Column: 2, Offset: 31}, OrigName: "main.Foo"},
	}
	if diff := cmp.Diff(wantEntries, entries); diff != "" {
		t.Errorf("Source map entries differ from expected (-want,+got):\n%s", diff)
	}
}

func writeHint(t *testing.T, w io.Writer, value any) {
	t.Helper()
	hint := Hint{}
	if err := hint.Pack(value); err != nil {
		t.Fatalf("Got: hint.Pack(%#v) returned error: %s. Want: no error.", value, err)
	}
	if _, err := hint.WriteTo(w); err != nil {
		t.Fatalf("Got: hint.WriteTo() returned error: %s. Want: no error.", err)
	}
}
