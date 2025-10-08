package tests_test

import "testing"

// Test_JSReservedWords uses testdata/reserved/main.go
// to test that JS reserved words can be used as labels, variable names, etc.
func Test_JSReservedWords(t *testing.T) { runOutputTest(t, `testdata`, `reserved`) }
