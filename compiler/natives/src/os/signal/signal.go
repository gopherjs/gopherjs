//go:build js

package signal

// Package signal is not implemented for GOARCH=js.

//gopherjs:replace
func signal_disable(uint32) {}

//gopherjs:replace
func signal_enable(uint32) {}

//gopherjs:replace
func signal_ignore(uint32) {}

//gopherjs:replace
func signal_recv() uint32 { return 0 }

//gopherjs:replace
func loop() {}
