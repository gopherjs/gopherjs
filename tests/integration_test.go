package tests_test

import "testing"

// Test_JSReservedWords uses testdata/reserved/main.go
// to test that JS reserved words can be used as labels, variable names, etc.
func Test_JSReservedWords(t *testing.T) { runOutputTest(t, `testdata`, `reserved`) }

// Test_JSSourceMap_Unminified uses testdata/jsSourceMap/main.go
// to test that the source map generated for the JS code is correct on unminified output.
func Test_JSSourceMap_Unminified(t *testing.T) { runOutputTest(t, `testdata`, `jsSourceMap`) }

// Test_JSSourceMap_Minified uses testdata/jsSourceMap/main.go
// to test that the source map generated for the JS code is correct on minified output.
func Test_JSSourceMap_Minified(t *testing.T) { runOutputTest(t, `testdata`, `jsSourceMap`, `-m`) }
