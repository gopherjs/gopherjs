//go:build js

package reflect

//gopherjs:purge Uses GC, stack, and funcLayout
func FuncLayout(t Type, rcvr Type) (frametype Type, argSize, retOffset uintptr, stack, gc, inReg, outReg []byte, ptrs bool)

//gopherjs:purge Uses internal information from map implementsion
func MapBucketOf(x, y Type) Type

//gopherjs:purge Uses internal information from map implementsion
func CachedBucketOf(m Type) Type

//gopherjs:purge Uses the byte name resolution
func FirstMethodNameBytes(t Type) *byte
