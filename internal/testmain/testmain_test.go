package testmain_test

import (
	gobuild "go/build"
	"go/token"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/gopherjs/gopherjs/build"
	"github.com/gopherjs/gopherjs/internal/srctesting"
	. "github.com/gopherjs/gopherjs/internal/testmain"
)

func TestScan(t *testing.T) {
	xctx := build.NewBuildContext("", nil)
	pkg, err := xctx.Import("github.com/gopherjs/gopherjs/internal/testmain/testdata/testpkg", "", 0)
	if err != nil {
		t.Fatalf("Failed to import package: %s", err)
	}

	fset := token.NewFileSet()

	got := TestMain{
		Package: pkg.Package,
		Context: pkg.InternalBuildContext(),
	}
	if err := got.Scan(fset); err != nil {
		t.Fatalf("Got: tm.Scan() returned error: %s. Want: no error.", err)
	}

	want := TestMain{
		TestMain: &TestFunc{Location: LocInPackage, Name: "TestMain"},
		Tests: []TestFunc{
			{Location: LocInPackage, Name: "TestXxx"},
			{Location: LocExternal, Name: "TestYyy"},
		},
		Benchmarks: []TestFunc{
			{Location: LocInPackage, Name: "BenchmarkXxx"},
			{Location: LocExternal, Name: "BenchmarkYyy"},
		},
		Fuzz: []TestFunc{
			{Location: LocInPackage, Name: "FuzzXxx"},
			{Location: LocExternal, Name: "FuzzYyy"},
		},
		Examples: []ExampleFunc{
			{Location: LocInPackage, Name: "ExampleXxx"},
			{Location: LocExternal, Name: "ExampleYyy", Output: "hello\n"},
		},
	}
	opts := cmp.Options{
		cmpopts.IgnoreFields(TestMain{}, "Package"), // Inputs.
		cmpopts.IgnoreFields(TestMain{}, "Context"),
	}
	if diff := cmp.Diff(want, got, opts...); diff != "" {
		t.Errorf("List of test function is different from expected (-want,+got):\n%s", diff)
	}
}

func TestSynthesize(t *testing.T) {
	pkg := &gobuild.Package{ImportPath: "foo/bar"}

	tests := []struct {
		descr   string
		tm      TestMain
		wantSrc string
	}{
		{
			descr: "all tests",
			tm: TestMain{
				Package: pkg,
				Tests: []TestFunc{
					{Location: LocInPackage, Name: "TestXxx"},
					{Location: LocExternal, Name: "TestYyy"},
				},
				Benchmarks: []TestFunc{
					{Location: LocInPackage, Name: "BenchmarkXxx"},
					{Location: LocExternal, Name: "BenchmarkYyy"},
				},
				Fuzz: []TestFunc{
					{Location: LocInPackage, Name: "FuzzXxx"},
					{Location: LocExternal, Name: "FuzzYyy"},
				},
				Examples: []ExampleFunc{
					{Location: LocInPackage, Name: "ExampleXxx", EmptyOutput: true},
					{Location: LocExternal, Name: "ExampleYyy", EmptyOutput: true},
				},
			},
			wantSrc: allTests,
		}, {
			descr: "testmain",
			tm: TestMain{
				Package:  pkg,
				TestMain: &TestFunc{Location: LocInPackage, Name: "TestMain"},
			},
			wantSrc: testmain,
		}, {
			descr: "import only",
			tm: TestMain{
				Package: pkg,
				Examples: []ExampleFunc{
					{Location: LocInPackage, Name: "ExampleXxx"},
				},
			},
			wantSrc: importOnly,
		},
	}

	for _, test := range tests {
		t.Run(test.descr, func(t *testing.T) {
			fset := token.NewFileSet()
			_, src, err := test.tm.Synthesize(fset)
			if err != nil {
				t.Fatalf("Got: tm.Synthesize() returned error: %s. Want: no error.", err)
			}
			got := srctesting.Format(t, fset, src)
			if diff := cmp.Diff(test.wantSrc, got); diff != "" {
				t.Errorf("Different _testmain.go source (-want,+got):\n%s", diff)
				t.Logf("Got source:\n%s", got)
			}
		})
	}
}

const allTests = `package main

import (
	"os"

	"testing"
	"testing/internal/testdeps"

	_test "foo/bar"
	_xtest "foo/bar_test"
)

var tests = []testing.InternalTest{
	{"TestXxx", _test.TestXxx},
	{"TestYyy", _xtest.TestYyy},
}

var benchmarks = []testing.InternalBenchmark{
	{"BenchmarkXxx", _test.BenchmarkXxx},
	{"BenchmarkYyy", _xtest.BenchmarkYyy},
}

var fuzzTargets = []testing.InternalFuzzTarget{
	{"FuzzXxx", _test.FuzzXxx},
	{"FuzzYyy", _xtest.FuzzYyy},
}

var examples = []testing.InternalExample{
	{"ExampleXxx", _test.ExampleXxx, "", false},
	{"ExampleYyy", _xtest.ExampleYyy, "", false},
}

func main() {
	m := testing.MainStart(testdeps.TestDeps{}, tests, benchmarks, fuzzTargets, examples)

	os.Exit(m.Run())
}
`

const testmain = `package main

import (
	"testing"
	"testing/internal/testdeps"

	_test "foo/bar"
)

var tests = []testing.InternalTest{}

var benchmarks = []testing.InternalBenchmark{}

var fuzzTargets = []testing.InternalFuzzTarget{}

var examples = []testing.InternalExample{}

func main() {
	m := testing.MainStart(testdeps.TestDeps{}, tests, benchmarks, fuzzTargets, examples)

	_test.TestMain(m)
}
`

const importOnly = `package main

import (
	"os"

	"testing"
	"testing/internal/testdeps"

	_ "foo/bar"
)

var tests = []testing.InternalTest{}

var benchmarks = []testing.InternalBenchmark{}

var fuzzTargets = []testing.InternalFuzzTarget{}

var examples = []testing.InternalExample{}

func main() {
	m := testing.MainStart(testdeps.TestDeps{}, tests, benchmarks, fuzzTargets, examples)

	os.Exit(m.Run())
}
`
