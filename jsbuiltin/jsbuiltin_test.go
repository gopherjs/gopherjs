package jsbuiltin

import (
	"testing"
//	"github.com/gopherjs/gopherjs/jsbuiltin"
)


func TestEncodeURI(t *testing.T) {
	data := map[string]string{
		"http://foo.com/?msg=Привет мир.": "http://foo.com/?msg=%D0%9F%D1%80%D0%B8%D0%B2%D0%B5%D1%82%20%D0%BC%D0%B8%D1%80.",
		"http://user:host@foo.com/": "http://user:host@foo.com/",
	}
	for url,expected := range data {
		result := EncodeURI(url)
		if result != expected {
			t.Fatalf("EncodeURI(%s) returned '%s', not '%s'", url, result, expected)
		}
	}
}

func TestDecodeURI(t *testing.T) {
	data := map[string]string{
		"http://foo.com/?msg=%D0%9F%D1%80%D0%B8%D0%B2%D0%B5%D1%82%20%D0%BC%D0%B8%D1%80.": "http://foo.com/?msg=Привет мир.",
		"http://user:host@foo.com/": "http://user:host@foo.com/",
	}
	for url,expected := range data {
		result := DecodeURI(url)
		if result != expected {
			t.Fatalf("DecodeURI(%s) returned '%s', not '%s'", url, result, expected)
		}
	}
}

func TestEncodeURIComponentn(t *testing.T) {
	data := map[string]string{
		"Привет мир.": "%D0%9F%D1%80%D0%B8%D0%B2%D0%B5%D1%82%20%D0%BC%D0%B8%D1%80.",
		"bar": "bar",
	}
	for url,expected := range data {
		result := EncodeURIComponent(url)
		if result != expected {
			t.Fatalf("EncodeURIComponent(%s) returned '%s', not '%s'", url, result, expected)
		}
	}
}

func TestDecodeURIComponentn(t *testing.T) {
	data := map[string]string{
		"%D0%9F%D1%80%D0%B8%D0%B2%D0%B5%D1%82%20%D0%BC%D0%B8%D1%80.": "Привет мир.",
		"bar": "bar",
	}
	for url,expected := range data {
		result := DecodeURIComponent(url)
		if result != expected {
			t.Fatalf("DecodeURIComponent(%s) returned '%s', not '%s'", url, result, expected)
		}
	}
}

func TestIsFinite(t *testing.T) {
	data := map[interface{}]bool {
		123:			true,
		-1.23:			true,
		5-2:			true,
		0:				true,
		"123":			true,
		"Hello":		false,
		"2005/12/12":	false,
	}
	for value,expected := range data {
		result := IsFinite(value)
		if result != expected {
			t.Fatalf("IsFinite(%s) returned %t, not %t", value, result, expected)
		}
	}
}

func TestIsNaN(t *testing.T) {
	data := map[interface{}]bool {
		123:			false,
		-1.23:			false,
		5-2:			false,
		0:				false,
		"123":			false,
		"Hello":		true,
		"2005/12/12":	true,
	}
	for value,expected := range data {
		result := IsNaN(value)
		if result != expected {
			t.Fatalf("IsNaN(%s) returned %t, not %t", value, result, expected)
		}
	}
}
