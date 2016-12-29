// +build js

package x509

import (
	"crypto"
	"crypto/ecdsa"
	"encoding/asn1"
	"encoding/base64"
	"errors"
	"math/big"
	"os"

	"github.com/gopherjs/gopherjs/js"
	"github.com/gopherjs/gopherjs/js/webcrypto"
)

var (
	errSignatureCheckFailed = errors.New("x509: signature check failed")
)

// This function overrides the original function in the standard library
func loadSystemRoots() (*CertPool, error) {
	return nil, errors.New("crypto/x509: system root pool is not available in GopherJS")
}

// This function overrides the original function in the standard library
func execSecurityRoots() (*CertPool, error) {
	return nil, os.ErrNotExist
}

func padLeft(buf []byte, length int) []byte {
	res := make([]byte, length)
	copy(res[length-len(buf):], buf)
	return res
}

// Converts a Go ecdsa.PublicKey to a JavaScript CryptoKey
func ecdsaPubKey2CryptoKey(pub *ecdsa.PublicKey) (*js.Object, error) {
	format := "jwk"
	// Json Web Key (https://tools.ietf.org/html/rfc7517, https://tools.ietf.org/html/rfc7518)

	order := len(pub.Params().P.Bytes())
	paddedX := padLeft(pub.X.Bytes(), order)
	paddedY := padLeft(pub.Y.Bytes(), order)
	jwkKey := js.M{
		"kty": "EC",
		"crv": pub.Params().Name,
		"x":   base64.RawURLEncoding.EncodeToString(paddedX),
		"y":   base64.RawURLEncoding.EncodeToString(paddedY),
	}

	algorithm := js.M{
		"name":       "ECDSA",
		"namedCurve": pub.Params().Name,
	}

	return webcrypto.SubtleCall("importKey", format, jwkKey, algorithm, false, []string{"verify"})
}

func webCryptoCheckSignature(algo SignatureAlgorithm, signed, signature []byte, publicKey crypto.PublicKey) error {
	switch pub := publicKey.(type) {
	case *ecdsa.PublicKey:
		cryptoKey, err := ecdsaPubKey2CryptoKey(pub)
		if err != nil {
			return err
		}
		algoName := ""
		switch algo {
		case ECDSAWithSHA1:
			algoName = "SHA-1"
		case ECDSAWithSHA256:
			algoName = "SHA-256"
		case ECDSAWithSHA384:
			algoName = "SHA-384"
		case ECDSAWithSHA512:
			algoName = "SHA-512"
		default:
			return ErrUnsupportedAlgorithm
		}

		// We have a ASN1 encoded signature and Web Crypto needs the concatenated padded valued of r and s
		sigStruct := new(ecdsaSignature)
		_, err = asn1.Unmarshal(signature, sigStruct)
		if err != nil {
			return err
		}

		order := len(pub.Params().P.Bytes())
		r := padLeft(sigStruct.R.Bytes(), order)
		s := padLeft(sigStruct.S.Bytes(), order)

		algorithm := js.M{
			"name": "ECDSA",
			"hash": js.M{"name": algoName},
		}

		res, err := webcrypto.SubtleCall("verify", algorithm, cryptoKey, append(r, s...), signed)

		if err != nil {
			return err
		}
		if !res.Bool() {
			return errSignatureCheckFailed
		}
		return nil
	}
	return ErrUnsupportedAlgorithm
}

//gopherjs:keep_overridden
// This function overrides the original function in the standard library
func checkSignature(algo SignatureAlgorithm, signed, signature []byte, publicKey crypto.PublicKey) error {
	err := webCryptoCheckSignature(algo, signed, signature, publicKey)
	if err == errSignatureCheckFailed {
		// WebCrypto said that the signature is not OK: fail
		return err
	}
	if err == nil {
		// WebCrypto said that the signature is OK: success
		return nil
	}
	// WebCrypto failed for another reason: fallback to the standard go implementation
	err = _gopherjs_overridden_checkSignature(algo, signed, signature, publicKey)
	return err
}
