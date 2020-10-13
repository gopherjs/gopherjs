// +build js
// +build go1.15

package poll

import (
	"errors"
)

var (
	ErrTimeout = errors.New("timeout")
)
