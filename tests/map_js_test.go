//go:build js && !wasm

package tests

import (
	"fmt"
	"reflect"
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

func mapCloneViaJS[M ~map[K]V, K comparable, V any](m M) M {
	mptr := &M{}
	cloned := js.Global.Get("Map").New(js.InternalObject(m))
	js.InternalObject(mptr).Call(`$set`, cloned)
	return *mptr
}

func mapCloneViaGo[M ~map[K]V, K comparable, V any](m M) M {
	mcopy := make(M, len(m))
	for k, v := range m {
		mcopy[k] = v
	}
	return mcopy
}

func BenchmarkMapClone(b *testing.B) {
	// Results from 2026/4/6 running go1.21.13 on darwin/arm64 with Node.js v20.9.0.
	//
	// The results show that mapCloneViaJS is faster than the mapCloneViaGo
	// after 5 key/value pairs. However, the speed is fast enough that for
	// smaller maps, mapCloneViaJS should still be fine. I'm guessing people
	// will clone larger maps that cloning tons of smaller maps, but if we need
	// to we can add a switch based on the size of the map.
	//
	// | size  | mapCloneViaGo (ns/op) | mapCloneViaJS (ns/op) | Go/JS (%) |
	// |------:|----------------------:|----------------------:|----------:|
	// |     0 |                 19.23 |                 96.00 |     20.03 |
	// |     1 |                 42.67 |                116.40 |     36.66 |
	// |     2 |                 82.14 |                138.10 |     59.48 |
	// |     3 |                111.60 |                167.90 |     66.47 |
	// |     5 |                232.50 |                245.80 |     94.59 |
	// |    10 |                469.60 |                408.70 |    114.90 |
	// |    50 |               2554.00 |               1712.00 |    149.18 |
	// |   100 |               5262.00 |               3630.00 |    144.96 |
	// |  1000 |              67155.00 |              38843.00 |    172.89 |
	// | 10000 |             956229.00 |             590539.00 |    161.92 |

	for _, size := range []int{0, 1, 2, 3, 5, 10, 50, 100, 1000, 10000} {
		m := make(map[string][]byte, size)
		for i := 0; i < size; i++ {
			key := fmt.Sprintf(`k%d`, i)
			value := []byte(fmt.Sprintf(`v%d`, i))
			m[key] = value
		}

		mcopy1 := (map[string][]byte)(nil)
		b.Run(fmt.Sprintf(`mapCloneViaJS(%d)`, size), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				mcopy1 = mapCloneViaJS(m)
			}
		})
		if !reflect.DeepEqual(m, mcopy1) {
			b.Errorf(`deep equal indicated mapCloneViaJS of size %d did not return expected copy`, size)
		}

		mcopy2 := (map[string][]byte)(nil)
		b.Run(fmt.Sprintf(`mapCloneViaGo(%d)`, size), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				mcopy2 = mapCloneViaGo(m)
			}
		})
		if !reflect.DeepEqual(m, mcopy2) {
			b.Errorf(`deep equal indicated mapCloneViaGo of size %d did not return expected copy`, size)
		}
	}
}

type myMap[K comparable] map[K]string

func (m myMap[K]) getOneKey() (key K, found bool) {
	for k := range m {
		return k, true
	}
	return
}

// TestMapCloneExtendedMap checks that the clone methods work on
// approxomate maps, i.e. `M ~map[K]V`, that have methods attached to it.
func TestMapCloneExtendedMap(t *testing.T) {
	m := myMap[int]{}
	key, found := m.getOneKey()
	if found {
		t.Errorf("expected a key to not be found but found a key of %v", key)
	}
	m[5] = "five"
	key, found = m.getOneKey()
	if !found {
		t.Errorf("expected a key to be found but it was not")
	}
	if key != 5 {
		t.Errorf("expected a key to be 5 but got %v", key)
	}
	m[2] = "two"
	m[42] = "answer"

	tests := []struct {
		name  string
		clone func(myMap[int]) myMap[int]
	}{
		{
			name:  `mapCloneViaJS`,
			clone: mapCloneViaJS[myMap[int]],
		}, {
			name:  `mapCloneViaGo`,
			clone: mapCloneViaGo[myMap[int]],
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mClone := test.clone(m)
			if want, got := len(m), len(mClone); got != want {
				t.Errorf("expected the cloned map to have length %d but got %d", want, got)
			}
			for key, want := range m {
				got, ok := mClone[key]
				if !ok {
					t.Errorf("expected the cloned map to have the key %v but it was not found", key)
				}
				if !ok {
					t.Errorf("expected the cloned map to have %v for key %v but got %v", want, key, got)
				}
			}
			key, found = mClone.getOneKey()
			if !found {
				t.Errorf("expected the cloned map to find a key but it did not")
			}
			if _, ok := m[key]; !ok {
				t.Errorf("expected the key %v from the cloned map to be a key found in the original but it was not", key)
			}
		})
	}
}
