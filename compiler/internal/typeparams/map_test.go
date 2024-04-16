package typeparams

import (
	"go/token"
	"go/types"
	"testing"
)

func TestInstanceMap(t *testing.T) {
	i1 := Instance{
		Object: types.NewTypeName(token.NoPos, nil, "i1", nil),
		TArgs: []types.Type{
			types.Typ[types.Int],
			types.Typ[types.Int8],
		},
	}
	i1clone := Instance{
		Object: i1.Object,
		TArgs: []types.Type{
			types.Typ[types.Int],
			types.Typ[types.Int8],
		},
	}

	i2 := Instance{
		Object: types.NewTypeName(token.NoPos, nil, "i2", nil), // Different pointer.
		TArgs: []types.Type{
			types.Typ[types.Int],
			types.Typ[types.Int8],
		},
	}
	i3 := Instance{
		Object: i1.Object,
		TArgs:  []types.Type{types.Typ[types.String]}, // Different type args.
	}

	_ = i1
	_ = i1clone
	_ = i3
	_ = i2

	m := InstanceMap[string]{}

	// Check operations on a missing key.
	t.Run("empty", func(t *testing.T) {
		if got := m.Has(i1); got {
			t.Errorf("Got: empty map contains %s. Want: empty map contains nothing.", i1)
		}
		if got := m.Get(i1); got != "" {
			t.Errorf("Got: getting missing key returned %q. Want: zero value.", got)
		}
		if got := m.Len(); got != 0 {
			t.Errorf("Got: empty map length %d. Want: 0.", got)
		}
		if got := m.Set(i1, "abc"); got != "" {
			t.Errorf("Got: setting a new key returned old value %q. Want: zero value", got)
		}
		if got := m.Len(); got != 1 {
			t.Errorf("Got: map length %d. Want: 1.", got)
		}
	})

	// Check operations on the existing key.
	t.Run("first key", func(t *testing.T) {
		if got := m.Set(i1, "def"); got != "abc" {
			t.Errorf(`Got: setting an existing key returned old value %q. Want: "abc".`, got)
		}
		if got := m.Len(); got != 1 {
			t.Errorf("Got: map length %d. Want: 1.", got)
		}
		if got := m.Has(i1); !got {
			t.Errorf("Got: set map key is reported as missing. Want: key present.")
		}
		if got := m.Get(i1); got != "def" {
			t.Errorf(`Got: getting set key returned %q. Want: "def"`, got)
		}
		if got := m.Get(i1clone); got != "def" {
			t.Errorf(`Got: getting set key returned %q. Want: "def"`, got)
		}
	})

	// Check for key collisions.
	t.Run("different object", func(t *testing.T) {
		if got := m.Has(i2); got {
			t.Errorf("Got: a new key %q is reported as present. Want: not present.", i2)
		}
		if got := m.Set(i2, "123"); got != "" {
			t.Errorf("Got: a new key %q overrode an old value %q. Want: zero value.", i2, got)
		}
		if got := m.Get(i2); got != "123" {
			t.Errorf(`Got: getting set key %q returned: %q. Want: "123"`, i2, got)
		}
		if got := m.Len(); got != 2 {
			t.Errorf("Got: map length %d. Want: 2.", got)
		}
	})
	t.Run("different tArgs", func(t *testing.T) {
		if got := m.Has(i3); got {
			t.Errorf("Got: a new key %q is reported as present. Want: not present.", i3)
		}
		if got := m.Set(i3, "456"); got != "" {
			t.Errorf("Got: a new key %q overrode an old value %q. Want: zero value.", i3, got)
		}
		if got := m.Get(i3); got != "456" {
			t.Errorf(`Got: getting set key %q returned: %q. Want: "456"`, i3, got)
		}
		if got := m.Len(); got != 3 {
			t.Errorf("Got: map length %d. Want: 3.", got)
		}
	})
}
