// +build js
// +build go1.13

package js

func (v Value) String() string {
	switch v.Type() {
	case TypeString:
		return v.internal().String()
	case TypeUndefined:
		return "<undefined>"
	case TypeNull:
		return "<null>"
	case TypeBoolean:
		return "<boolean: " + v.internal().String() + ">"
	case TypeNumber:
		return "<number: " + v.internal().String() + ">"
	case TypeSymbol:
		return "<symbol>"
	case TypeObject:
		return "<object>"
	case TypeFunction:
		return "<function>"
	default:
		panic("bad type")
	}
}

// CopyBytesToGo copies bytes from the Uint8Array src to dst.
// It returns the number of bytes copied, which will be the minimum of the lengths of src and dst.
// CopyBytesToGo panics if src is not an Uint8Array.
func CopyBytesToGo(dst []byte, src Value) int {
	return copy(dst, src.internal().Interface().([]byte))
}

// func CopyBytesToGo(dst []byte, src Value) int {
// 	s := make([]byte, src.Get("byteLength").Int())
// 	a := TypedArrayOf(s)
// 	a.Call("set", src)
// 	a.Release()
// 	return copy(dst, s)
// }

// CopyBytesToJS copies bytes from src to the Uint8Array dst.
// It returns the number of bytes copied, which will be the minimum of the lengths of src and dst.
// CopyBytesToJS panics if dst is not an Uint8Array.
func CopyBytesToJS(dst Value, src []byte) int {
	return copy(dst.internal().Interface().([]byte), src)
}

// func CopyBytesToJS(dst Value, src []byte) int {
// 	n := dst.Get("byteLength").Int()
// 	if n > len(src) {
// 		n = len(src)
// 	}
// 	a := TypedArrayOf(src[:n])
// 	dst.Call("set", a)
// 	a.Release()
// 	return n
// }
