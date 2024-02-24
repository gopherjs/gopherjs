//go:build js
// +build js

package netip

type Addr struct {
	addr uint128
	// Unlike the upstream, we store the string directly instead of trying to
	// use internal/intern package for optimization.
	z string
}

var (
	// Sentinel values for different zones. \x00 character makes it unlikely for
	// the sentinel value to clash with any real-life IPv6 zone index.
	z0    = ""
	z4    = "\x00ipv4"
	z6noz = "\x00ipv6noz"
)

func (ip Addr) Zone() string {
	if ip.z == z4 || ip.z == z6noz {
		return ""
	}
	return ip.z
}

func (ip Addr) WithZone(zone string) Addr {
	if !ip.Is6() {
		return ip
	}
	if zone == "" {
		ip.z = z6noz
		return ip
	}
	ip.z = zone
	return ip
}
