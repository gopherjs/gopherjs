// +build js

package x509

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
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

// Converts a Go crypto.PublicKey to a JavaScript CryptoKey
func pubKey2CryptoKey(publicKey crypto.PublicKey) (*js.Object, error) {
	format := "jwk"
	// Json Web Key (https://tools.ietf.org/html/rfc7517, https://tools.ietf.org/html/rfc7518)
	var jwkKey, algorithm js.M

	switch pub := publicKey.(type) {
	case *ecdsa.PublicKey:
		order := len(pub.Params().P.Bytes())
		paddedX := padLeft(pub.X.Bytes(), order)
		paddedY := padLeft(pub.Y.Bytes(), order)
		jwkKey = js.M{
			"kty": "EC",
			"crv": pub.Params().Name,
			"x":   base64.RawURLEncoding.EncodeToString(paddedX),
			"y":   base64.RawURLEncoding.EncodeToString(paddedY),
		}

		algorithm = js.M{
			"name":       "ECDSA",
			"namedCurve": pub.Params().Name,
		}
	case *rsa.PublicKey:
		jwkKey = js.M{
			"kty": "RSA",
			"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
			"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
		}

		algorithm = js.M{"name": "RSA"}
	default:
		return nil, ErrUnsupportedAlgorithm
	}

	return webcrypto.SubtleCall("importKey", format, jwkKey, algorithm, false, []string{"verify"})
}

func webCryptoCheckSignature(algo SignatureAlgorithm, signed, signature []byte, publicKey crypto.PublicKey) error {
	cryptoKey, err := pubKey2CryptoKey(publicKey)
	if err != nil {
		return err
	}
	var algoName, hashAlgoName string
	var webCryptoSig []byte

	switch algo {
	case ECDSAWithSHA1, SHA1WithRSA:
		hashAlgoName = "SHA-1"
	case ECDSAWithSHA256, SHA256WithRSA:
		hashAlgoName = "SHA-256"
	case ECDSAWithSHA384, SHA384WithRSA:
		hashAlgoName = "SHA-384"
	case ECDSAWithSHA512, SHA512WithRSA:
		hashAlgoName = "SHA-512"
	default:
		return ErrUnsupportedAlgorithm
	}

	switch pub := publicKey.(type) {
	case *ecdsa.PublicKey:
		algoName = "ECDSA"

		// We have a ASN1 encoded signature and Web Crypto needs the concatenated padded valued of r and s
		sigStruct := new(ecdsaSignature)
		_, err = asn1.Unmarshal(signature, sigStruct)
		if err != nil {
			return err
		}
		order := len(pub.Params().P.Bytes())
		r := padLeft(sigStruct.R.Bytes(), order)
		s := padLeft(sigStruct.S.Bytes(), order)
		webCryptoSig = append(r, s...)

	case *rsa.PublicKey:
		algoName = "RSA"
		webCryptoSig = signature

	default:
		return ErrUnsupportedAlgorithm
	}

	algorithm := js.M{
		"name": algoName,
		"hash": js.M{"name": hashAlgoName},
	}

	res, err := webcrypto.SubtleCall("verify", algorithm, cryptoKey, webCryptoSig, signed)

	if err != nil {
		return err
	}
	if !res.Bool() {
		return errSignatureCheckFailed
	}
	return nil
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
