//go:build js
// +build js

package netip

import (
	"fmt"

	"internal/intern"
)

func MkAddr(u Uint128, z any) Addr {
	switch z := z.(type) {
	case *intern.Value:
		return Addr{u, z.Get().(string)}
	case string:
		return Addr{u, z}
	default:
		panic(fmt.Errorf("unexpected type %T of the z argument"))
	}
}
