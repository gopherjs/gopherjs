// +build js

package syscall

import (
	"bytes"
)

var warningPrinted = false
var lineBuffer []byte

func printWarning() {
	if !warningPrinted {
		println("warning: system calls not available, see https://github.com/gopherjs/gopherjs/blob/master/doc/syscalls.md")
	}
	warningPrinted = true
}

func printToConsole(b []byte) {
	lineBuffer = append(lineBuffer, b...)
	for {
		i := bytes.IndexByte(lineBuffer, '\n')
		if i == -1 {
			break
		}
		println(string(lineBuffer[:i]))
		lineBuffer = lineBuffer[i+1:]
	}
}
