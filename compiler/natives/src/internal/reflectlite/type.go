//go:build js

package reflectlite

//gopherjs:purge Unused in reflectlite, abi.ArrayType is used instead
type arrayType struct{}

//gopherjs:purge Unused in reflectlite, abi.ChanType is used instead
type chanType struct{}

//gopherjs:purge Unused in reflectlite, abi.MapType is used instead
type mapType struct{}

//gopherjs:purge Unused in reflectlite, abi.PtrType is used instead
type ptrType struct{}

//gopherjs:purge Unused in reflectlite, abi.SliceType is used instead
type sliceType struct{}

//gopherjs:replace
func (t rtype) Comparable() bool {
	return t.common().Comparable()
}

//gopherjs:replace
func (t rtype) String() string {
	return t.common().String()
}
