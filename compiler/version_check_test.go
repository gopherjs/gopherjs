package compiler

import (
	"runtime"
	"strings"
	"testing"
)

func TestGoRelease(t *testing.T) {
	t.Run("goroot", func(t *testing.T) {
		got := GoRelease(runtime.GOROOT())
		want := runtime.Version()
		if got != want {
			t.Fatalf("Got: goRelease(%q) returned %q. Want %s.", runtime.GOROOT(), got, want)
		}
	})

	t.Run("fallback", func(t *testing.T) {
		const goroot = "./invalid goroot"
		got := GoRelease(goroot)
		if got == "" {
			t.Fatalf("Got: goRelease(%q) returned \"\". Want: a Go version.", goroot)
		}
		if !strings.HasSuffix(Version, "+"+got) {
			t.Fatalf("Got: goRelease(%q) returned %q. Want: a fallback to GopherJS version suffix %q.", goroot, got, Version)
		}
	})
}
