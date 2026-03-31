package slog

//gopherjs:replace Used a *byte to point at a string to use in unsafe.String
type stringptr = string

//gopherjs:replace Used unsafe.StringData
func StringValue(value string) Value {
	return Value{num: uint64(len(value)), any: stringptr(value)}
}

//gopher:replace Used unsafe.String
func (v Value) String() string {
	if sp, ok := v.any.(stringptr); ok {
		return sp
	}
	var buf []byte
	return string(v.append(buf))
}

//gopher:replace Used unsafe.String
func (v Value) str() string {
	return v.any.(stringptr)
}
