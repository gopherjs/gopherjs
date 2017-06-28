// +build js

// Package webcrypto provides primitives used to call the Web Cryptography API
// https://www.w3.org/TR/WebCryptoAPI/
package webcrypto

import (
	"errors"

	"github.com/gopherjs/gopherjs/js"
)

// JavaScript Crypto interface, or nil if it was not found
var Crypto *js.Object = getCrypto()

// JavaScript SubtleCrypto interface, or nil if it was not found
var SubtleCrypto *js.Object = getSubtle()

var (
	ErrWebCryptoInterfaceNotFound = errors.New("crypto: SubtleCrypto interface not found")
	ErrWebCryptoFunctionNotFound  = errors.New("crypto: function not found")
	ErrWebCryptoDataError         = errors.New("crypto: DataError (WebCryptoAPI)")
	ErrWebCryptoNotSupportedError = errors.New("crypto: NotSupportedError (WebCryptoAPI)")
	ErrWebCryptoOperationError    = errors.New("crypto: OperationError (WebCryptoAPI)")
)

func getCrypto() *js.Object {
	crypto := js.Global.Get("crypto")
	if crypto == js.Undefined {
		crypto = js.Global.Get("msCrypto") // for IE11
		if crypto == js.Undefined {
			return nil
		}
	}
	return crypto
}

func getSubtle() *js.Object {
	if Crypto == nil {
		return nil
	}
	subtle := Crypto.Get("subtle")
	if subtle == js.Undefined {
		subtle = Crypto.Get("webkitSubtle") // for Safari
		if subtle == js.Undefined {
			return nil
		}
	}
	return subtle
}

//  Uncomment this init function to be able to run tests in the browser:
//  You can then run
//    gopherjs test -c crypto/<module>
//  And load the generated javascript code in a browser

// func init() {
// 	os.Args = append(os.Args, "-test.v")
// 	os.Args = append(os.Args, "-test.bench=.")
// }

// SubtleCall calls the object's method with the given arguments assuming that it returns a promise, wait for it to be settled, and returns the result/error.
func SubtleCall(method string, args ...interface{}) (res *js.Object, err error) {
	if SubtleCrypto == nil {
		return nil, ErrWebCryptoInterfaceNotFound
	}

	// First check that the method exists (if it doesn't, Call() would panic, so it's better to check first)
	if SubtleCrypto.Get(method) == js.Undefined {
		return nil, ErrWebCryptoFunctionNotFound
	}

	resCh := make(chan *js.Object, 1)
	failCh := make(chan *js.Object, 1)

	defer func() {
		switch exc := recover().(type) {
		case *js.Error:
			err = exc
		case nil:
			break
		default:
			panic(exc)
		}
	}()

	promise := SubtleCrypto.Call(method, args...)
	promise.Call(
		"then",
		func(o *js.Object) { resCh <- o },
		func(err *js.Object) { failCh <- err })

	select {
	case jsres := <-resCh:
		return jsres, nil
	case jserr := <-failCh:
		switch jserr.Get("name").String() {
		case "DataError":
			return nil, ErrWebCryptoDataError
		case "NotSupportedError":
			return nil, ErrWebCryptoNotSupportedError
		case "OperationError":
			return nil, ErrWebCryptoOperationError
		default:
			return nil, &js.Error{Object: jserr}
		}
	}
}

func GetBytes(obj *js.Object) []byte {
	array := js.Global.Get("Uint8Array").New(obj)
	return array.Interface().([]byte)
}
