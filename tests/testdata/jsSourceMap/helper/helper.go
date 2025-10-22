package helper

import "github.com/gopherjs/gopherjs/js"

func DoGoThing() string {
	return js.Global.Call(`doJSThing`).String()
}
