// Package js provides functions for interacting with native JavaScript APIs. Calls to these functions are treated specially by GopherJS and translated directly to their corresponding JavaScript syntax.
// The types Object and Any are containers for JavaScript objects and are no real citizens of the Go world, e.g. they do not implement interface{} and you cannot type assert them. The type Any may also contain Go values, but it provides no methods for accessing its content. It is only useful for passing it to functions of this package. The relation between interface{} (Go world), Object (JS world) and Any looks like this:
//
//  +------------------------------------+
//  |                                    |
//  |  +-------------+  +-------------+  |
//  |  |             |  |             |  |
//  |  | interface{} |  |  js.Object  |  |
//  |  |             |  |             |  |
//  |  +-------------+  +-------------+  |
//  |                                    |
//  |               js.Any               |
//  |                                    |
//  +------------------------------------+
//
// Type conversions between Go types and JavaScript types are performed automatically according to the following table:
//
//  | Go type                        | Conversions to interface{} | JavaScript type   |
//  | ------------------------------ | -------------------------- | ----------------- |
//  | bool                           | bool                       | Boolean           |
//  | int?, uint?, float?            | float64                    | Number            |
//  | string                         | string                     | String            |
//  | [?]int8                        | []int8                     | Int8Array         |
//  | [?]int16                       | []int16                    | Int16Array        |
//  | [?]int32, [?]int               | []int                      | Int32Array        |
//  | [?]uint8                       | []uint8                    | Uint8Array        |
//  | [?]uint16                      | []uint16                   | Uint16Array       |
//  | [?]uint32, [?]uint, [?]uintptr | []uint                     | Uint32Array       |
//  | [?]float32                     | []float32                  | Float32Array      |
//  | [?]float64                     | []float64                  | Float64Array      |
//  | all other slices and arrays    | []interface{}              | Array             |
//  | functions                      | func(...js.Any) js.Object  | Function          |
//  | time.Time                      | time.Time                  | Date              |
//  | -                              | *js.DOMNode                | instanceof Node   |
//  | maps, structs                  | map[string]interface{}     | instanceof Object |
//
// Additionally, for a struct containing a js.Object field, only the content of the field will be passed to JavaScript and vice versa.
package js

// Object is a container for a native JavaScript object. Calls to its methods are treated specially by GopherJS and translated directly to their JavaScript syntax. Nil is equal to JavaScript's "null".
// Object can not be used as a map key and it can only be converted to Any, but no other interface type, including interface{}. This means that you can't pass an Object obj to fmt.Println(obj), but using the println(obj) built-in is possible.
type Object interface {

	// Get returns the object's property with the given key.
	Get(key string) Object

	// Set assigns the value to the object's property with the given key.
	Set(key string, value Any)

	// Delete removes the object's property with the given key.
	Delete(key string)

	// Get returns the object's "length" property, converted to int.
	Length() int

	// Index returns the i'th element of an array.
	Index(i int) Object

	// SetIndex sets the i'th element of an array.
	SetIndex(i int, value Any)

	// Call calls the object's method with the given name.
	Call(name string, args ...Any) Object

	// Invoke calls the object itself. This will fail if it is not a function.
	Invoke(args ...Any) Object

	// New creates a new instance of this type object. This will fail if it not a function (constructor).
	New(args ...Any) Object

	// Bool returns the object converted to bool according to JavaScript type conversions.
	Bool() bool

	// String returns the object converted to string according to JavaScript type conversions.
	String() string

	// Int returns the object converted to int according to JavaScript type conversions (parseInt).
	Int() int

	// Int64 returns the object converted to int64 according to JavaScript type conversions (parseInt).
	Int64() int64

	// Uint64 returns the object converted to uint64 according to JavaScript type conversions (parseInt).
	Uint64() uint64

	// Float returns the object converted to float64 according to JavaScript type conversions (parseFloat).
	Float() float64

	// Interface returns the object converted to interface{}. See GopherJS' README for details.
	Interface() interface{}

	// Unsafe returns the object as an uintptr, which can be converted via unsafe.Pointer. Not intended for public use.
	Unsafe() uintptr
}

// Any may contain a JavaScript object or some Go value. Any can not be used as a map key and it can not be converted to some other interface type, including interface{}.
type Any interface{}

// DOMNode is used for encapsulating DOM nodes in a proper Go value. It is not feasible to convert a DOM node into a map[string]interface{}.
type DOMNode struct {
	Object
}

// Error encapsulates JavaScript errors. Those are turned into a Go panic and may be rescued, giving an *Error that holds the JavaScript error object.
type Error struct {
	Object
}

// Error returns the message of the encapsulated JavaScript error object.
func (err *Error) Error() string {
	return "JavaScript error: " + err.Get("message").String()
}

// Stack returns the stack property of the encapsulated JavaScript error object.
func (err *Error) Stack() string {
	return err.Get("stack").String()
}

// Global gives JavaScript's global object ("window" for browsers and "GLOBAL" for Node.js). Set this to a mock for testing with pure Go.
var Global Object

// This gives the value of JavaScript's "this" keyword. It can be used when passing Go functions to JavaScript as callbacks. Set this to a mock for testing with pure Go.
var This Object

// Arguments gives the value of JavaScript's "arguments" keyword. It can be used when passing Go functions to JavaScript as callbacks. Set this to a mock for testing with pure Go.
var Arguments []Object

// Module gives the value of the "module" variable set by Node.js. Hint: Set a module export with 'js.Module.Get("exports").Set("exportName", ...)'.
var Module Object

// Undefined gives the JavaScript value "undefined".
var Undefined Object

// Debugger gets compiled to JavaScript's "debugger;" statement.
func Debugger() {}

// InternalObject returns the internal JavaScript object that represents i. Not intended for public use.
func InternalObject(i interface{}) Object {
	return nil
}

// Keys returns the keys of the given JavaScript object.
func Keys(o Object) []string {
	if o == Undefined || o == nil {
		return nil
	}
	a := Global.Get("Object").Call("keys", o)
	s := make([]string, a.Length())
	for i := 0; i < a.Length(); i++ {
		s[i] = a.Index(i).String()
	}
	return s
}

// MakeWrapper creates a JavaScript object which has wrappers for the exported methods of i. This can be used to provide getter and setter methods for Go fields to JavaScript.
func MakeWrapper(i interface{}) Object {
	v := InternalObject(i)
	o := Global.Get("Object").New()
	methods := v.Get("constructor").Get("methods")
	for i := 0; i < methods.Length(); i++ {
		m := methods.Index(i)
		if m.Get("pkg").String() != "" { // not exported
			continue
		}
		o.Set(m.Get("name").String(), func(args ...Any) Object {
			return v.Call(m.Get("prop").String(), args...)
		})
	}
	return o
}

// M is a simple map type. It is intended as a shorthand for JavaScript objects (before conversion).
type M map[string]Any

// S is a simple slice type. It is intended as a shorthand for JavaScript arrays (before conversion).
type S []Any

func init() {
	// avoid dead code elimination
	e := Error{}
	n := DOMNode{}
	_, _ = e, n
}
