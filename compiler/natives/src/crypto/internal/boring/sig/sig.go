//go:build js

package sig

// Setting to no-op
//
//gopherjs:replace
func BoringCrypto() {}

// Setting to no-op
//
//gopherjs:replace
func FIPSOnly() {}

// Setting to no-op
//
//gopherjs:replace
func StandardCrypto() {}
