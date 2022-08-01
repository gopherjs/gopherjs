package testpkg_test

import (
	"fmt"
	"testing"
)

func TestYyy(t *testing.T) {}

func BenchmarkYyy(b *testing.B) {}

func FuzzYyy(f *testing.F) { f.Skip() }

func ExampleYyy() {
	fmt.Println("hello") // Output: hello
}
