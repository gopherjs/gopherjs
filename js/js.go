// Package js provides functions for interacting with native JavaScript APIs. Calls to these functions are treated specially by GopherJS and translated directly to their JavaScript syntax.
//
// Type conversions between Go types and JavaScript types are performed automatically according to the table below. Types not listed are passed through. The second column denotes the types that are used when converting to `interface{}`.
//
//  | Go types                       | Go interface type              | JavaScript type |
//  | ------------------------------ | ------------------------------ | --------------- |
//  | bool                           | bool                           | Boolean         |
//  | int?, uint?, float?            | float64                        | Number          |
//  | string                         | string                         | String          |
//  | [?]int8                        | []int8                         | Int8Array       |
//  | [?]int16                       | []int16                        | Int16Array      |
//  | [?]int32, [?]int               | []int                          | Int32Array      |
//  | [?]uint8                       | []uint8                        | Int8Array       |
//  | [?]uint16                      | []uint16                       | Int16Array      |
//  | [?]uint32, [?]uint, [?]uintptr | []uint                         | Int32Array      |
//  | [?]float32                     | []float32                      | Float32Array    |
//  | [?]float64                     | []float64                      | Float64Array    |
//  | all other slices and arrays    | []interface{}                  | Array           |
//  | maps                           | map[string]interface{}         | Object          |
//  | functions                      | func(...interface{}) js.Object | Function        |
//  | time.Time                      | time.Time                      | Date            |
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

	// Str returns the object converted to string according to JavaScript type conversions. Does not implement fmt.Stringer on purpose.
	Str() string

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

// Error encapsulates JavaScript errors. Those are turned into a Go panic and may be rescued, giving an *Error that holds the JavaScript error object.
type Error struct {
	Object
}

// Error returns the message of the encapsulated JavaScript error object.
func (err *Error) Error() string {
	return "JavaScript error: " + err.Get("message").Str()
}

// Global gives JavaScript's global object ("window" for browsers and "GLOBAL" for Node.js). Set this to a mock for testing with pure Go.
var Global Object

// This gives the value of JavaScript's "this" keyword. It can be used when passing Go functions to JavaScript as callbacks. Set this to a mock for testing with pure Go.
var This Object

// Returns the internal JavaScript object that represents i. Not intended for public use.
func InternalObject(i interface{}) Object {
	return nil
}
