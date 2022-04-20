//go:build js

package tls

import (
	"context"
	"runtime"
	"testing"
)

// Same as upstream, except we check for GOARCH=ecmascript instead of wasm.
// This override can be removed after https://github.com/golang/go/pull/51827
// is available in the upstream (likely in Go 1.19).
func TestServerHandshakeContextCancellation(t *testing.T) {
	c, s := localPipe(t)
	ctx, cancel := context.WithCancel(context.Background())
	unblockClient := make(chan struct{})
	defer close(unblockClient)
	go func() {
		cancel()
		<-unblockClient
		_ = c.Close()
	}()
	conn := Server(s, testConfig)
	// Initiates server side handshake, which will block until a client hello is read
	// unless the cancellation works.
	err := conn.HandshakeContext(ctx)
	if err == nil {
		t.Fatal("Server handshake did not error when the context was canceled")
	}
	if err != context.Canceled {
		t.Errorf("Unexpected server handshake error: %v", err)
	}
	if runtime.GOARCH == "ecmascript" {
		t.Skip("conn.Close does not error as expected when called multiple times on WASM")
	}
	err = conn.Close()
	if err == nil {
		t.Error("Server connection was not closed when the context was canceled")
	}
}

// Same as upstream, except we check for GOARCH=ecmascript instead of wasm.
// This override can be removed after https://github.com/golang/go/pull/51827
// is available in the upstream (likely in Go 1.19).
func TestClientHandshakeContextCancellation(t *testing.T) {
	c, s := localPipe(t)
	ctx, cancel := context.WithCancel(context.Background())
	unblockServer := make(chan struct{})
	defer close(unblockServer)
	go func() {
		cancel()
		<-unblockServer
		_ = s.Close()
	}()
	cli := Client(c, testConfig)
	// Initiates client side handshake, which will block until the client hello is read
	// by the server, unless the cancellation works.
	err := cli.HandshakeContext(ctx)
	if err == nil {
		t.Fatal("Client handshake did not error when the context was canceled")
	}
	if err != context.Canceled {
		t.Errorf("Unexpected client handshake error: %v", err)
	}
	if runtime.GOARCH == "ecmascript" {
		t.Skip("conn.Close does not error as expected when called multiple times on WASM")
	}
	err = cli.Close()
	if err == nil {
		t.Error("Client connection was not closed when the context was canceled")
	}
}
