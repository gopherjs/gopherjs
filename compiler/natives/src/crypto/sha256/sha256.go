// +build js

package sha256

import (
	"errors"

	"github.com/gopherjs/gopherjs/js"
	"github.com/gopherjs/gopherjs/js/webcrypto"
)

var errCryptoWrongLength = errors.New("crypto: Unexpected hash length")

func webCryptoSum256(data []byte) (res [Size]byte, err error) {
	jsArray, err := webcrypto.SubtleCall("digest", js.M{"name": "SHA-256"}, data)
	if err != nil {
		return res, err
	}

	slice := webcrypto.GetBytes(jsArray)
	if len(slice) != Size {
		return res, errCryptoWrongLength
	}
	copy(res[:], webcrypto.GetBytes(jsArray))
	return res, err
}

//gopherjs:keep_overridden
// This function overrides the original function in the standard library
func Sum256(data []byte) [Size]byte {
	res, err := webCryptoSum256(data)
	if err != nil {
		// The WebCryptoAPI call failed: fallback by calling the implementation from
		// the Go standard library
		return _gopherjs_overridden_Sum256(data)
	}
	return res
}
