//go:build js && wasm

package http

func init() {
	// Use standard transport with fake networking under tests. Although GopherJS
	// supports "real" http.Client implementations using Fetch or XMLHttpRequest
	// APIs, tests also need to start local web servers, which is not supported
	// for those APIs.
	// TODO(nevkontakte): We could test our real implementations if we mock out
	// browser APIs and redirect them to the fake networking stack, but this is
	// not easy.
	jsFetchMissing = true
	DefaultTransport = &Transport{}
}
