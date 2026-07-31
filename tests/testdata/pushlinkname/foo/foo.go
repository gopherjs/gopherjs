package foo

import _ "unsafe"

type Foo struct{ name string }

func (f *Foo) String() string { return f.name }

func setName1(f *Foo, name string) { f.name = name }

func getName1(f Foo) string { return f.name }

//go:linkname setName2 github.com/gopherjs/gopherjs/tests/testdata/pushlinkname/pushLink.pushSetName
func setName2(f *Foo, name string) { f.name = name }

//go:linkname getName2 github.com/gopherjs/gopherjs/tests/testdata/pushlinkname/pushLink.pushGetName
func getName2(f Foo) string { return f.name }

func setName3(f *Foo, name string) { f.name = name }

func getName3(f Foo) string { return f.name }

//go:linkname setName3 github.com/gopherjs/gopherjs/tests/testdata/pushlinkname/anotherPushLink.pushSetName
//go:linkname getName3 github.com/gopherjs/gopherjs/tests/testdata/pushlinkname/anotherPushLink.pushGetName
