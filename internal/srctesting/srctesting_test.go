package srctesting

import "testing"

func TestFixture(t *testing.T) {
	f := New(t)

	const src1 = `package foo
	type X int
	`
	_, foo := f.Check("pkg/foo", f.Parse("foo.go", src1))

	if !foo.Complete() {
		t.Fatalf("Got: incomplete package pkg/foo: %s. Want: complete package.", foo)
	}

	const src2 = `package bar
	import "pkg/foo"
	func Fun() foo.X { return 0 }
	`

	// Should type check successfully with dependency on pkg/foo.
	_, bar := f.Check("pkg/bar", f.Parse("bar.go", src2))

	if !bar.Complete() {
		t.Fatalf("Got: incomplete package pkg/bar: %s. Want: complete package.", foo)
	}
}
