// +build js
// +build go1.15

package reflect_test

import (
	"fmt"
	. "reflect"
	"strings"
	"testing"
)

func TestConvertNaNs(t *testing.T) {
	t.Skip()
}

func shouldPanic(expect string, f func()) {
	defer func() {
		r := recover()
		if r == nil {
			panic("did not panic")
		}
		if expect != "" {
			var s string
			switch r := r.(type) {
			case string:
				s = r
			case *ValueError:
				s = r.Error()
			default:
				panic(fmt.Sprintf("panicked with unexpected type %T", r))
			}
			if !strings.HasPrefix(s, "reflect") {
				panic(`panic string does not start with "reflect": ` + s)
			}
			if !strings.Contains(s, expect) {
				//	panic(`panic string does not contain "` + expect + `": ` + s)
			}
		}
	}()
	f()
}
