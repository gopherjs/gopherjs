//go:build js && !wasm

package tests

import (
	"testing"

	"github.com/gopherjs/gopherjs/js"
)

func Test_MapWrapper(t *testing.T) {
	// This tests that various map types, and a map as a function argument and return,
	// wrap and unwrap correctly.
	type Dummy struct {
		Msg string
	}

	type StructWithMap struct {
		StringMap map[string]string
		IntMap    map[int]int
		DummyMap  map[string]*Dummy
		MapFunc   func(map[string]string) map[string]string
	}

	dummyMap := map[string]*Dummy{"key": {Msg: "value"}}
	swm := &StructWithMap{
		StringMap: map[string]string{"key": "value"},
		IntMap:    map[int]int{1: 2},
		DummyMap:  dummyMap,
		MapFunc: func(m map[string]string) map[string]string {
			return m
		},
	}
	swmWrapper := js.MakeFullWrapper(swm)
	swmUnwrapped := swmWrapper.Interface().(*StructWithMap)
	mapFuncArg := map[string]string{"key2": "value2"}

	if got := swmWrapper.Get("StringMap").Get("key").String(); got != swm.StringMap["key"] {
		t.Errorf("StringMap Got: %s, Want: %s", got, swm.StringMap["key"])
	}
	if got := swmWrapper.Get("IntMap").Get("1").Int(); got != swm.IntMap[1] {
		t.Errorf("IntMap Got: %d, Want: %d", got, swm.IntMap[1])
	}
	if got := swmWrapper.Get("DummyMap").Get("key").Get("Msg").String(); got != swm.DummyMap["key"].Msg {
		t.Errorf("DummyMap Got: %s, Want: %s", got, swm.DummyMap["key"].Msg)
	}
	if got := swmWrapper.Call("MapFunc", mapFuncArg).Get("key2").String(); got != mapFuncArg["key2"] {
		t.Errorf("MapFunc Got: %s, Want: %s", got, mapFuncArg["key2"])
	}

	if got := swmUnwrapped.StringMap["key"]; got != swm.StringMap["key"] {
		t.Errorf("Unwrapped StringMap Got: %s, Want: %s", got, swm.StringMap["key"])
	}
	if got := swmUnwrapped.IntMap[1]; got != swm.IntMap[1] {
		t.Errorf("Unwrapped IntMap Got: %d, Want: %d", got, swm.IntMap[1])
	}
	if got := swmUnwrapped.DummyMap["key"].Msg; got != swm.DummyMap["key"].Msg {
		t.Errorf("Unwrapped DummyMap Got: %s, Want: %s", got, swm.DummyMap["key"].Msg)
	}
	if got := swmUnwrapped.MapFunc(mapFuncArg)["key2"]; got != swm.MapFunc(mapFuncArg)["key2"] {
		t.Errorf("Unwrapped MapFunc Got: %s, Want: %s", got, swm.MapFunc(mapFuncArg)["key2"])
	}
}

func Test_MapStructObjectWrapper(t *testing.T) {
	// This tests that maps work as expected when wrapping a Struct with js.Object field containing a map.
	// js.Object fields' content should be passed to JS, so this is basically doubly-wrapping a map in two structs.

	stringMap := map[string]string{"key": "value"}

	// You cannot wrap a map directly, so put it in a struct.
	type StructWithMap struct {
		Map map[string]string
	}

	swm := &StructWithMap{Map: stringMap}
	swmWrapped := js.MakeFullWrapper(swm)

	// Now that map is wrapped in a struct, wrap the js.object in *another* struct.
	type StructWithObject struct {
		Wrappedswm *js.Object // This Object contains StructWithMap.
	}

	swo := &StructWithObject{Wrappedswm: swmWrapped}
	swoWrapper := js.MakeFullWrapper(swo)
	swmUnwrapped := swoWrapper.Interface().(*StructWithObject)

	// Using "Get("Map")" shows that the first wrapping was unchanged.
	if got := swoWrapper.Get("Wrappedswm").Get("Map").Get("key").String(); got != stringMap["key"] {
		t.Errorf("Wrapped Wrappedswm value Got: %s, Want: %s", got, stringMap["key"])
	}

	if got := swmUnwrapped.Wrappedswm.Get("Map").Get("key").String(); got != stringMap["key"] {
		t.Errorf("Unwrapped Wrappedswm value Got: %s, Want: %s", got, stringMap["key"])
	}
}

func Test_MapEmbeddedObject(t *testing.T) {
	o := js.Global.Get("JSON").Call("parse", `{"props": {"one": 1, "two": 2}}`)

	type data struct {
		*js.Object
		Props map[string]int `js:"props"`
	}

	d := data{Object: o}
	if d.Props["one"] != 1 {
		t.Errorf("key 'one' value Got: %d, Want: %d", d.Props["one"], 1)
	}
	if d.Props["two"] != 2 {
		t.Errorf("key 'two' value Got: %d, Want: %d", d.Props["two"], 2)
	}
}
