//go:build js

package unsafeheader

// Slice and String is Go's runtime representations which is different
// from GopherJS's runtime representations. By purging these types,
// it will prevent failures in JS where the code compiles fine but
// expects there to be a constructor which doesn't exist when casting
// from GopherJS's representation into Go's representation.

//gopherjs:purge
type Slice struct{}

//gopherjs:purge
type String struct{}
