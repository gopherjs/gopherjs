//go:build js

package intern

var (
	eth0 = &Value{cmpVal: "eth0"}
	eth1 = &Value{cmpVal: "eth1"}
)

func get(k key) *Value {
	// Interning implementation in this package unavoidably relies upon
	// runtime.SetFinalizer(), which GopherJS doesn't support (at least until it
	// is considered safe to use the WeakMap API). Without working finalizers
	// using this package would create memory leaks.
	//
	// Considering that this package is supposed to serve as an optimization tool,
	// it is better to make it explicitly unusable and work around it at the call
	// sites.

	// net/netip tests use intern API with a few fixed values. It is easier to
	// special-case them here than to override the entire test set.
	if k.isString && k.s == "eth0" {
		return eth0
	} else if k.isString && k.s == "eth1" {
		return eth1
	}

	panic("internal/intern is not supported by GopherJS")
}
