package testpkg

import "testing"

func TestXxx(t *testing.T) {}

func BenchmarkXxx(b *testing.B) {}

func FuzzXxx(f *testing.F) { f.Skip() }

func ExampleXxx() {}

func TestMain(m *testing.M) { m.Run() }
