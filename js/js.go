// Package js provides functions for interacting with native JavaScript APIs. Calls to these functions are treated specially by GopherJS and translated directly to their corresponding JavaScript syntax.
//
// Type conversions between Go types and JavaScript types are performed automatically according to the following table:
//
//  | Go type               | JavaScript type       | Conversions back to interface{} |
//  | --------------------- | --------------------- | ------------------------------- |
//  | bool                  | Boolean               | bool                            |
//  | integers and floats   | Number                | float64                         |
//  | string                | String                | string                          |
//  | []int8                | Int8Array             | []int8                          |
//  | []int16               | Int16Array            | []int16                         |
//  | []int32, []int        | Int32Array            | []int                           |
//  | []uint8               | Uint8Array            | []uint8                         |
//  | []uint16              | Uint16Array           | []uint16                        |
//  | []uint32, []uint      | Uint32Array           | []uint                          |
//  | []float32             | Float32Array          | []float32                       |
//  | []float64             | Float64Array          | []float64                       |
//  | all other slices      | Array                 | []interface{}                   |
//  | arrays                | see slice type        | see slice type                  |
//  | functions             | Function              | func(...interface{}) js.Object  |
//  | time.Time             | Date                  | time.Time                       |
//  | -                     | instanceof Node       | js.Object                       |
//  | maps, structs         | instanceof Object     | map[string]interface{}          |
//
// Additionally, for a struct containing a js.Object field, only the content of the field will be passed to JavaScript and vice versa.
package js

// Object is a container for a native JavaScript object. Calls to its methods are treated specially by GopherJS and translated directly to their JavaScript syntax. Nil is equal to JavaScript's "null". Object can not be used as a map key.
type Object interface {

	// Get returns the object's property with the given key.
	Get(key string) Object

	// Set assigns the value to the object's property with the given key.
	Set(key string, value interface{})

	// Delete removes the object's property with the given key.
	Delete(key string)

	// Get returns the object's "length" property, converted to int.
	Length() int

	// Index returns the i'th element of an array.
	Index(i int) Object

	// SetIndex sets the i'th element of an array.
	SetIndex(i int, value interface{})

	// Call calls the object's method with the given name.
	Call(name string, args ...interface{}) Object

	// Invoke calls the object itself. This will fail if it is not a function.
	Invoke(args ...interface{}) Object

	// New creates a new instance of this type object. This will fail if it not a function (constructor).
	New(args ...interface{}) Object

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

type container struct{ Object }

func (c *container) Get(key string) Object                        { return c.Object.Get(key) }
func (c *container) Set(key string, value interface{})            { c.Object.Set(key, value) }
func (c *container) Delete(key string)                            { c.Object.Delete(key) }
func (c *container) Length() int                                  { return c.Object.Length() }
func (c *container) Index(i int) Object                           { return c.Object.Index(i) }
func (c *container) SetIndex(i int, value interface{})            { c.Object.SetIndex(i, value) }
func (c *container) Call(name string, args ...interface{}) Object { return c.Object.Call(name, args...) }
func (c *container) Invoke(args ...interface{}) Object            { return c.Object.Invoke(args...) }
func (c *container) New(args ...interface{}) Object               { return c.Object.New(args...) }
func (c *container) Bool() bool                                   { return c.Object.Bool() }
func (c *container) String() string                               { return c.Object.String() }
func (c *container) Int() int                                     { return c.Object.Int() }
func (c *container) Int64() int64                                 { return c.Object.Int64() }
func (c *container) Uint64() uint64                               { return c.Object.Uint64() }
func (c *container) Float() float64                               { return c.Object.Float() }
func (c *container) Interface() interface{}                       { return c.Object.Interface() }
func (c *container) Unsafe() uintptr                              { return c.Object.Unsafe() }

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
		o.Set(m.Get("name").String(), func(args ...Object) Object {
			paramTypes := m.Get("typ").Get("params")
			internalizedArgs := make([]interface{}, paramTypes.Length())
			for i := range internalizedArgs {
				internalizedArgs[i] = Global.Call("$internalize", args[i], paramTypes.Index(i))
			}
			result := v.Call(m.Get("prop").String(), internalizedArgs...)
			resultTypes := m.Get("typ").Get("results")
			switch resultTypes.Length() {
			case 0:
				return nil
			case 1:
				return Global.Call("$externalize", result, resultTypes.Index(0))
			default:
				a := Global.Get("Array").New(resultTypes.Length())
				for i := 0; i < resultTypes.Length(); i++ {
					a.SetIndex(i, Global.Call("$externalize", result.Index(i), resultTypes.Index(i)))
				}
				return a
			}
		})
	}
	return o
}

// NewArrayBuffer creates a JavaScript ArrayBuffer from a byte slice.
func NewArrayBuffer(b []byte) Object {
	slice := InternalObject(b)
	offset := slice.Get("$offset").Int()
	length := slice.Get("$length").Int()
	return slice.Get("$array").Get("buffer").Call("slice", offset, offset+length)
}

// M is a simple map type. It is intended as a shorthand for JavaScript objects (before conversion).
type M map[string]interface{}

// S is a simple slice type. It is intended as a shorthand for JavaScript arrays (before conversion).
type S []interface{}

func init() {
	// avoid dead code elimination
	c := container{}
	e := Error{}
	_, _ = c, e
}
