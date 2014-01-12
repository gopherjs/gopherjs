package js

type Object interface {
	Get(key string) Object
	Set(key string, value interface{})
	Length() int
	Index(i int) Object
	Call(name string, args ...interface{}) Object
	Invoke(args ...interface{}) Object
	New(args ...interface{}) Object
	Bool() bool
	String() string
	Int() int
	Float() float64
	Interface() interface{}
	IsUndefined() bool
	IsNull() bool
}

func Global(name string) Object {
	return nil
}
