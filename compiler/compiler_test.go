package compiler

import (
	"bytes"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/tools/go/loader"
)

func TestOrder(t *testing.T) {
	fileA := `
package foo

var Avar = "a"

type Atype struct{}

func Afunc() int {
	var varA = 1
	var varB = 2
	return varA+varB
}
`

	fileB := `
package foo

var Bvar = "b"

type Btype struct{}

func Bfunc() int {
	var varA = 1
	var varB = 2
	return varA+varB
}
`
	files := []source{{"fileA.go", []byte(fileA)}, {"fileB.go", []byte(fileB)}}

	compare(t, "foo", files, false)
	compare(t, "foo", files, true)
}

func compare(t *testing.T, path string, sourceFiles []source, minify bool) {
	outputNormal, err := compile(path, sourceFiles, minify)
	if err != nil {
		t.Fatal(err)
	}

	// reverse the array
	for i, j := 0, len(sourceFiles)-1; i < j; i, j = i+1, j-1 {
		sourceFiles[i], sourceFiles[j] = sourceFiles[j], sourceFiles[i]
	}

	outputReversed, err := compile(path, sourceFiles, minify)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(string(outputNormal), string(outputReversed)); diff != "" {
		t.Errorf("files in different order produce different JS:\n%s", diff)
	}
}

type source struct {
	name     string
	contents []byte
}

func compile(path string, sourceFiles []source, minify bool) ([]byte, error) {
	conf := loader.Config{}
	conf.Fset = token.NewFileSet()
	conf.ParserMode = parser.ParseComments

	context := build.Default // make a copy of build.Default
	conf.Build = &context
	conf.Build.BuildTags = []string{"js"}

	var astFiles []*ast.File
	for _, sourceFile := range sourceFiles {
		astFile, err := parser.ParseFile(conf.Fset, sourceFile.name, sourceFile.contents, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		astFiles = append(astFiles, astFile)
	}
	conf.CreateFromFiles(path, astFiles...)
	prog, err := conf.Load()
	if err != nil {
		return nil, err
	}

	archiveCache := map[string]*Archive{}
	var importContext *ImportContext
	importContext = &ImportContext{
		Packages: make(map[string]*types.Package),
		Import: func(path string) (*Archive, error) {
			// find in local cache
			if a, ok := archiveCache[path]; ok {
				return a, nil
			}

			pi := prog.Package(path)
			importContext.Packages[path] = pi.Pkg

			// compile package
			a, err := Compile(path, pi.Files, prog.Fset, importContext, minify)
			if err != nil {
				return nil, err
			}
			archiveCache[path] = a
			return a, nil
		},
	}

	a, err := importContext.Import(path)
	if err != nil {
		return nil, err
	}
	b, err := renderPackage(a)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func renderPackage(archive *Archive) ([]byte, error) {
	selection := make(map[*Decl]struct{})
	for _, d := range archive.Declarations {
		selection[d] = struct{}{}
	}

	buf := &bytes.Buffer{}

	if err := WritePkgCode(archive, selection, goLinknameSet{}, false, &SourceMapFilter{Writer: buf}); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
