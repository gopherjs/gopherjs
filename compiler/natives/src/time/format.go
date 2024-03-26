//go:build js
// +build js

package time

// copied and replaced for go1.20 temporarily without generics.
func atoi(sAny any) (x int, err error) {
	s := asBytes(sAny)
	neg := false
	if len(s) > 0 && (s[0] == '-' || s[0] == '+') {
		neg = s[0] == '-'
		s = s[1:]
	}
	q, remStr, err := leadingInt(s)
	rem := []byte(remStr)
	x = int(q)
	if err != nil || len(rem) > 0 {
		return 0, atoiError
	}
	if neg {
		x = -x
	}
	return x, nil
}

// copied and replaced for go1.20 temporarily without generics.
func isDigit(sAny any, i int) bool {
	s := asBytes(sAny)
	if len(s) <= i {
		return false
	}
	c := s[i]
	return '0' <= c && c <= '9'
}

// copied and replaced for go1.20 temporarily without generics.
func parseNanoseconds(sAny any, nbytes int) (ns int, rangeErrString string, err error) {
	value := asBytes(sAny)
	if !commaOrPeriod(value[0]) {
		err = errBad
		return
	}
	if nbytes > 10 {
		value = value[:10]
		nbytes = 10
	}
	if ns, err = atoi(value[1:nbytes]); err != nil {
		return
	}
	if ns < 0 {
		rangeErrString = "fractional second"
		return
	}
	scaleDigits := 10 - nbytes
	for i := 0; i < scaleDigits; i++ {
		ns *= 10
	}
	return
}

// copied and replaced for go1.20 temporarily without generics.
func leadingInt(sAny any) (x uint64, rem string, err error) {
	s := asBytes(sAny)
	i := 0
	for ; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			break
		}
		if x > 1<<63/10 {
			return 0, rem, errLeadingInt
		}
		x = x*10 + uint64(c) - '0'
		if x > 1<<63 {
			return 0, rem, errLeadingInt
		}
	}
	return x, string(s[i:]), nil
}
