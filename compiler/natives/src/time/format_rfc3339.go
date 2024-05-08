//go:build js
// +build js

package time

import "errors"

// added for go1.20 temporarily without generics.
func asBytes(s any) []byte {
	switch t := s.(type) {
	case []byte:
		return t
	case string:
		return []byte(t)
	default:
		panic(errors.New(`unexpected type passed to asBytes, expected string or []bytes`))
	}
}

// copied and replaced for go1.20 temporarily without generics.
func parseRFC3339(sAny any, local *Location) (Time, bool) {
	s := asBytes(sAny)
	ok := true
	parseUint := func(s []byte, min, max int) (x int) {
		for _, c := range s {
			if c < '0' || '9' < c {
				ok = false
				return min
			}
			x = x*10 + int(c) - '0'
		}
		if x < min || max < x {
			ok = false
			return min
		}
		return x
	}

	if len(s) < len("2006-01-02T15:04:05") {
		return Time{}, false
	}
	year := parseUint(s[0:4], 0, 9999)
	month := parseUint(s[5:7], 1, 12)
	day := parseUint(s[8:10], 1, daysIn(Month(month), year))
	hour := parseUint(s[11:13], 0, 23)
	min := parseUint(s[14:16], 0, 59)
	sec := parseUint(s[17:19], 0, 59)
	if !ok || !(s[4] == '-' && s[7] == '-' && s[10] == 'T' && s[13] == ':' && s[16] == ':') {
		return Time{}, false
	}
	s = s[19:]

	var nsec int
	if len(s) >= 2 && s[0] == '.' && isDigit(s, 1) {
		n := 2
		for ; n < len(s) && isDigit(s, n); n++ {
		}
		nsec, _, _ = parseNanoseconds(s, n)
		s = s[n:]
	}

	t := Date(year, Month(month), day, hour, min, sec, nsec, UTC)
	if len(s) != 1 || s[0] != 'Z' {
		if len(s) != len("-07:00") {
			return Time{}, false
		}
		hr := parseUint(s[1:3], 0, 23)
		mm := parseUint(s[4:6], 0, 59)
		if !ok || !((s[0] == '-' || s[0] == '+') && s[3] == ':') {
			return Time{}, false
		}
		zoneOffset := (hr*60 + mm) * 60
		if s[0] == '-' {
			zoneOffset *= -1
		}
		t.addSec(-int64(zoneOffset))

		if _, offset, _, _, _ := local.lookup(t.unixSec()); offset == zoneOffset {
			t.setLoc(local)
		} else {
			t.setLoc(FixedZone("", zoneOffset))
		}
	}
	return t, true
}
