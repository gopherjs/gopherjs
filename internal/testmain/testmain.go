package testmain

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/build"
	"go/doc"
	"go/parser"
	"go/token"
	"path"
	"sort"
	"strings"
	"text/template"
	"unicode"
	"unicode/utf8"

	"golang.org/x/tools/go/buildutil"
)

// FuncLocation describes whether a test function is in-package or external
// (i.e. in the xxx_test package).
type FuncLocation uint8

const (
	// LocUnknown is the default, invalid value of the PkgType.
	LocUnknown FuncLocation = iota
	// LocInPackage is an in-package test.
	LocInPackage
	// LocExternal is an external test (i.e. in the xxx_test package).
	LocExternal
)

func (tl FuncLocation) String() string {
	switch tl {
	case LocInPackage:
		return "_test"
	case LocExternal:
		return "_xtest"
	default:
		return "<unknown>"
	}
}

// TestFunc describes a single test/benchmark/fuzz function in a package.
type TestFunc struct {
	Location FuncLocation // Where the function is defined.
	Name     string       // Function name.
}

// ExampleFunc describes an example.
type ExampleFunc struct {
	Location    FuncLocation // Where the function is defined.
	Name        string       // Function name.
	Output      string       // Expected output.
	Unordered   bool         // Output is allowed to be unordered.
	EmptyOutput bool         // Whether the output is expected to be empty.
}

// Executable returns true if the example function should be executed with tests.
func (ef ExampleFunc) Executable() bool {
	return ef.EmptyOutput || ef.Output != ""
}

// TestMain is a helper type responsible for generation of the test main package.
type TestMain struct {
	Package    *build.Package
	Context    *build.Context
	Tests      []TestFunc
	Benchmarks []TestFunc
	Fuzz       []TestFunc
	Examples   []ExampleFunc
	TestMain   *TestFunc
}

// Scan package for tests functions.
func (tm *TestMain) Scan(fset *token.FileSet) error {
	if err := tm.scanPkg(fset, tm.Package.TestGoFiles, LocInPackage); err != nil {
		return err
	}
	if err := tm.scanPkg(fset, tm.Package.XTestGoFiles, LocExternal); err != nil {
		return err
	}
	return nil
}

func (tm *TestMain) scanPkg(fset *token.FileSet, files []string, loc FuncLocation) error {
	for _, name := range files {
		srcPath := path.Join(tm.Package.Dir, name)
		f, err := buildutil.OpenFile(tm.Context, srcPath)
		if err != nil {
			return fmt.Errorf("failed to open source file %q: %w", srcPath, err)
		}
		defer f.Close()
		parsed, err := parser.ParseFile(fset, srcPath, f, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("failed to parse %q: %w", srcPath, err)
		}

		if err := tm.scanFile(parsed, loc); err != nil {
			return err
		}
	}
	return nil
}

func (tm *TestMain) scanFile(f *ast.File, loc FuncLocation) error {
	for _, d := range f.Decls {
		n, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if n.Recv != nil {
			continue
		}
		name := n.Name.String()
		switch {
		case isTestMain(n):
			if tm.TestMain != nil {
				return errors.New("multiple definitions of TestMain")
			}
			tm.TestMain = &TestFunc{
				Location: loc,
				Name:     name,
			}
		case isTest(name, "Test"):
			tm.Tests = append(tm.Tests, TestFunc{
				Location: loc,
				Name:     name,
			})
		case isTest(name, "Benchmark"):
			tm.Benchmarks = append(tm.Benchmarks, TestFunc{
				Location: loc,
				Name:     name,
			})
		case isTest(name, "Fuzz"):
			tm.Fuzz = append(tm.Fuzz, TestFunc{
				Location: loc,
				Name:     name,
			})
		}
	}

	ex := doc.Examples(f)
	sort.Slice(ex, func(i, j int) bool { return ex[i].Order < ex[j].Order })
	for _, e := range ex {
		tm.Examples = append(tm.Examples, ExampleFunc{
			Location:    loc,
			Name:        "Example" + e.Name,
			Output:      e.Output,
			Unordered:   e.Unordered,
			EmptyOutput: e.EmptyOutput,
		})
	}

	return nil
}

// Synthesize main package for the tests.
func (tm *TestMain) Synthesize(fset *token.FileSet) (*build.Package, *ast.File, error) {
	buf := &bytes.Buffer{}
	if err := testmainTmpl.Execute(buf, tm); err != nil {
		return nil, nil, fmt.Errorf("failed to generate testmain source for package %s: %w", tm.Package.ImportPath, err)
	}
	src, err := parser.ParseFile(fset, "_testmain.go", buf, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse testmain source for package %s: %w", tm.Package.ImportPath, err)
	}
	pkg := &build.Package{
		ImportPath: tm.Package.ImportPath + ".testmain",
		Name:       "main",
		GoFiles:    []string{"_testmain.go"},
	}
	return pkg, src, nil
}

func (tm *TestMain) hasTests(loc FuncLocation, executableOnly bool) bool {
	if tm.TestMain != nil && tm.TestMain.Location == loc {
		return true
	}
	// Tests, Benchmarks and Fuzz targets are always executable.
	all := []TestFunc{}
	all = append(all, tm.Tests...)
	all = append(all, tm.Benchmarks...)

	for _, t := range all {
		if t.Location == loc {
			return true
		}
	}

	for _, e := range tm.Examples {
		if e.Location == loc && (e.Executable() || !executableOnly) {
			return true
		}
	}
	return false
}

// ImportTest returns true if in-package test package needs to be imported.
func (tm *TestMain) ImportTest() bool { return tm.hasTests(LocInPackage, false) }

// ImportXTest returns true if external test package needs to be imported.
func (tm *TestMain) ImportXTest() bool { return tm.hasTests(LocExternal, false) }

// ExecutesTest returns true if in-package test package has executable tests.
func (tm *TestMain) ExecutesTest() bool { return tm.hasTests(LocInPackage, true) }

// ExecutesXTest returns true if external package test package has executable tests.
func (tm *TestMain) ExecutesXTest() bool { return tm.hasTests(LocExternal, true) }

// isTestMain tells whether fn is a TestMain(m *testing.M) function.
func isTestMain(fn *ast.FuncDecl) bool {
	if fn.Name.String() != "TestMain" ||
		fn.Type.Results != nil && len(fn.Type.Results.List) > 0 ||
		fn.Type.Params == nil ||
		len(fn.Type.Params.List) != 1 ||
		len(fn.Type.Params.List[0].Names) > 1 {
		return false
	}
	ptr, ok := fn.Type.Params.List[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	// We can't easily check that the type is *testing.M
	// because we don't know how testing has been imported,
	// but at least check that it's *M or *something.M.
	if name, ok := ptr.X.(*ast.Ident); ok && name.Name == "M" {
		return true
	}
	if sel, ok := ptr.X.(*ast.SelectorExpr); ok && sel.Sel.Name == "M" {
		return true
	}
	return false
}

// isTest tells whether name looks like a test (or benchmark, according to prefix).
// It is a Test (say) if there is a character after Test that is not a lower-case letter.
// We don't want TesticularCancer.
func isTest(name, prefix string) bool {
	if !strings.HasPrefix(name, prefix) {
		return false
	}
	if len(name) == len(prefix) { // "Test" is ok
		return true
	}
	rune, _ := utf8.DecodeRuneInString(name[len(prefix):])
	return !unicode.IsLower(rune)
}

var testmainTmpl = template.Must(template.New("main").Parse(`
package main

import (
{{if not .TestMain}}
	"os"
{{end}}
	"testing"
	"testing/internal/testdeps"

{{if .ImportTest}}
	{{if .ExecutesTest}}_test{{else}}_{{end}} {{.Package.ImportPath | printf "%q"}}
{{end -}}
{{- if .ImportXTest -}}
	{{if .ExecutesXTest}}_xtest{{else}}_{{end}} {{.Package.ImportPath | printf "%s_test" | printf "%q"}}
{{end}}
)

var tests = []testing.InternalTest{
{{- range .Tests}}
	{"{{.Name}}", {{.Location}}.{{.Name}}},
{{- end}}
}

var benchmarks = []testing.InternalBenchmark{
{{- range .Benchmarks}}
	{"{{.Name}}", {{.Location}}.{{.Name}}},
{{- end}}
}

var fuzzTargets = []testing.InternalFuzzTarget{
{{- range .Fuzz}}
	{"{{.Name}}", {{.Location}}.{{.Name}}},
{{- end}}
}

var examples = []testing.InternalExample{
{{- range .Examples }}
{{- if .Executable }}
	{"{{.Name}}", {{.Location}}.{{.Name}}, {{.Output | printf "%q"}}, {{.Unordered}}},
{{- end }}
{{- end }}
}

func main() {
	m := testing.MainStart(testdeps.TestDeps{}, tests, benchmarks, fuzzTargets, examples)
{{with .TestMain}}
	{{.Location}}.{{.Name}}(m)
{{else}}
	os.Exit(m.Run())
{{end -}}
}

`))
