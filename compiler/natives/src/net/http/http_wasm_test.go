//go:build js && wasm
// +build js,wasm

package http

func init() {
	if DefaultTransport == nil {
		panic("Expected DefaultTransport to be initialized before init() is run!")
	}
	if _, ok := DefaultTransport.(noTransport); ok && useFakeNetwork {
		// NodeJS doesn't provide Fetch API by default, but wasm version of standard
		// library has a fake network implementation that can be used to execute
		// tests. If we are running under tests, use of fake networking is requested
		// and no actual transport is found, use the fake that's implemented by
		// `Transport`.
		DefaultTransport = &Transport{}
	}
}
