package js

// Object is a container for a native JavaScript object. Calls to its methods are treated specially by GopherJS and translated directly to their JavaScript syntax.
type Object interface {

	// Get returns the object's property with the given name.
	Get(name string) Object

	// Set assigns the value to the object's property with the given name.
	Set(name string, value interface{})

	// Get returns the object's "length" property, converted to int.
	Length() int

	// Index returns the i'th element of an array.
	Index(i int) Object

	// SetIndex sets the i'th element of an array.
	SetIndex(i int, value interface{})

	// Call calls the object's method with the given name.
	Call(name string, args ...interface{}) Object

	// Invoce calls the object itself. This will fail if it is not a function.
	Invoke(args ...interface{}) Object

	// New creates a new instance of this type object. This will fail if it not a function (constructor).
	New(args ...interface{}) Object

	// Bool returns the object converted to bool according to JavaScript type conversions.
	Bool() bool

	// String returns the object converted to string according to JavaScript type conversions.
	String() string

	// Int returns the object converted to int according to JavaScript type conversions (parseInt).
	Int() int

	// Float returns the object converted to float64 according to JavaScript type conversions (parseFloat).
	Float() float64

	// Interface returns the object converted to interface{}. See GopherJS' README for details.
	Interface() interface{}

	// IsUndefined returns true if the object is the JavaScript value "undefine".
	IsUndefined() bool

	// IsNull returns true if the object is the JavaScript value "null".
	IsNull() bool
}

// Global returns the JavaScript global with the given name.
func Global(name string) Object {
	return nil // GopherJS will not use this function body
}

// This returns the value of JavaScript's "this" keyword. It can be used when passing Go functions to JavaScript as callbacks.
func This() Object {
	return nil // GopherJS will not use this function body
}
