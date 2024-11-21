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
		TArgs: []types.Type{ // Different type args, same number.
			types.Typ[types.Int],
			types.Typ[types.Int],
		},
	}
	i4 := Instance{
		Object: i1.Object,
		TArgs: []types.Type{ // This hash matches i3's hash.
			types.Typ[types.String],
			types.Typ[types.String],
		},
	}
	i5 := Instance{
		Object: i1.Object,
		TArgs:  []types.Type{}, // This hash matches i3's hash.
	}

	m := InstanceMap[string]{}

	// Check operations on a missing key.
	t.Run("empty", func(t *testing.T) {
		if got, want := m.String(), `{}`; got != want {
			t.Errorf("Got: empty map string %q. Want: map string %q.", got, want)
		}
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
		if got, want := m.String(), `{{type i1 int, int8}:abc}`; got != want {
			t.Errorf("Got: map string %q. Want: map string %q.", got, want)
		}
		if got, want := m.Keys(), []Instance{i1}; !keysMatch(got, want) {
			t.Errorf("Got: map keys %v. Want: [i1].", got)
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
		if got, want := m.String(), `{{type i1 int, int8}:def}`; got != want {
			t.Errorf("Got: map string %q. Want: map string %q.", got, want)
		}
		if got, want := m.Keys(), []Instance{i1}; !keysMatch(got, want) {
			t.Errorf("Got: map keys %v. Want: [i1].", got)
		}
	})

	// Check for key collisions with different object pointer.
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

	// Check for collisions with different type arguments and different hash.
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

	// Check for collisions with different type arguments, same hash, count.
	t.Run("different tArgs hash", func(t *testing.T) {
		if got := m.Has(i4); got {
			t.Errorf("Got: a new key %q is reported as present. Want: not present.", i3)
		}
		if got := m.Set(i4, "789"); got != "" {
			t.Errorf("Got: a new key %q overrode an old value %q. Want: zero value.", i3, got)
		}
		if got := m.Get(i4); got != "789" {
			t.Errorf(`Got: getting set key %q returned: %q. Want: "789"`, i3, got)
		}
		if got := m.Len(); got != 4 {
			t.Errorf("Got: map length %d. Want: 4.", got)
		}
	})

	// Check for collisions with different type arguments and same hash, but different count.
	t.Run("different tArgs count", func(t *testing.T) {
		if got := m.Has(i5); got {
			t.Errorf("Got: a new key %q is reported as present. Want: not present.", i3)
		}
		if got := m.Set(i5, "ghi"); got != "" {
			t.Errorf("Got: a new key %q overrode an old value %q. Want: zero value.", i3, got)
		}
		if got := m.Get(i5); got != "ghi" {
			t.Errorf(`Got: getting set key %q returned: %q. Want: "ghi"`, i3, got)
		}
		if got := m.Len(); got != 5 {
			t.Errorf("Got: map length %d. Want: 5.", got)
		}
		if got, want := m.String(), `{{type i1 int, int8}:def, {type i1 int, int}:456, {type i1 string, string}:789, {type i1 }:ghi, {type i2 int, int8}:123}`; got != want {
			t.Errorf("Got: map string %q. Want: map string %q.", got, want)
		}
		if got, want := m.Keys(), []Instance{i1, i2, i3, i4, i5}; !keysMatch(got, want) {
			t.Errorf("Got: map keys %v. Want: [i1, i2, i3, i4, i5].", got)
		}
	})

	// Check an existing entry can be deleted.
	t.Run("delete existing", func(t *testing.T) {
		if got := m.Delete(i3); !got {
			t.Errorf("Got: deleting existing key %q returned not deleted. Want: found and deleted.", i3)
		}
		if got := m.Len(); got != 4 {
			t.Errorf("Got: map length %d. Want: 4.", got)
		}
		if got := m.Has(i3); got {
			t.Errorf("Got: a deleted key %q is reported as present. Want: not present.", i3)
		}
		if got, want := m.Keys(), []Instance{i1, i2, i4, i5}; !keysMatch(got, want) {
			t.Errorf("Got: map keys %v. Want: [i1, i2, i4, i5].", got)
		}
	})

	// Check deleting an existing entry has no effect.
	t.Run("delete already deleted", func(t *testing.T) {
		if got := m.Delete(i3); got {
			t.Errorf("Got: deleting not present key %q returned as deleted. Want: not found.", i3)
		}
		if got := m.Len(); got != 4 {
			t.Errorf("Got: map length %d. Want: 4.", got)
		}
		if got, want := m.Keys(), []Instance{i1, i2, i4, i5}; !keysMatch(got, want) {
			t.Errorf("Got: map keys %v. Want: [i1, i2, i4, i5].", got)
		}
	})

	// Check adding back a deleted value works (should fill hole in bucket).
	t.Run("set deleted key", func(t *testing.T) {
		if got := m.Set(i3, "jkl"); got != "" {
			t.Errorf("Got: a new key %q overrode an old value %q. Want: zero value.", i3, got)
		}
		if got := m.Len(); got != 5 {
			t.Errorf("Got: map length %d. Want: 5.", got)
		}
		if got, want := m.Keys(), []Instance{i1, i2, i3, i4, i5}; !keysMatch(got, want) {
			t.Errorf("Got: map keys %v. Want: [i1, i2, i3, i4, i5].", got)
		}
	})

	// Check deleting while iterating over the map.
	t.Run("deleting while iterating", func(t *testing.T) {
		notSeen := []Instance{i1, i2, i3, i4, i5}
		seen := []Instance{}
		kept := []Instance{}
		var skipped Instance
		m.Iterate(func(key Instance, value string) {
			// update seen and not seen
			seen = append(seen, key)
			i := keyAt(notSeen, key)
			if i < 0 {
				t.Fatalf(`Got: failed to find current key %q in not seen. Want: it to be not seen yet.`, key)
			}
			notSeen = append(notSeen[:i], notSeen[i+1:]...)

			if len(seen) == 3 {
				// delete the first seen key, the current key, and an unseen key
				if got := m.Delete(seen[0]); !got {
					t.Errorf("Got: deleting seen key %q returned not deleted. Want: found and deleted.", seen[0])
				}
				if got := m.Delete(key); !got {
					t.Errorf("Got: deleting current key %q returned not deleted. Want: found and deleted.", key)
				}
				skipped = notSeen[0] // skipped has not yet been seen so it should not be iterated over
				if got := m.Delete(skipped); !got {
					t.Errorf("Got: deleting not seen key %q returned not deleted. Want: found and deleted.", skipped)
				}
				kept = append(kept, seen[1], notSeen[1])
			}
		})

		if got := len(seen); got != 4 {
			t.Errorf("Got: seen %d keys. Want: 4.", got)
		}
		if got := len(notSeen); got != 1 {
			t.Errorf("Got: seen %d keys. Want: 1.", got)
		}
		if got := keyAt(notSeen, skipped); got != 0 {
			t.Errorf("Got: a deleted unseen key %q was not the skipped key %q. Want: it to be skipped.", notSeen[0], skipped)
		}
		if got := m.Len(); got != 2 {
			t.Errorf("Got: map length %d. Want: 2.", got)
		}
		if got := m.Keys(); !keysMatch(got, kept) {
			t.Errorf("Got: map keys %v did not match kept keys. Want: %v.", got, kept)
		}
	})
}

func TestNilInstanceMap(t *testing.T) {
	i1 := Instance{
		Object: types.NewTypeName(token.NoPos, nil, "i1", nil),
		TArgs: []types.Type{
			types.Typ[types.Int],
			types.Typ[types.Int8],
		},
	}

	var m *InstanceMap[string]
	if got, want := m.String(), `{}`; got != want {
		t.Errorf("Got: nil map string %q. Want: map string %q.", got, want)
	}
	if got := m.Has(i1); got {
		t.Errorf("Got: nil map contains %s. Want: nil map contains nothing.", i1)
	}
	if got := m.Get(i1); got != "" {
		t.Errorf("Got: missing key returned %q. Want: zero value.", got)
	}
	if got := m.Len(); got != 0 {
		t.Errorf("Got: nil map length %d. Want: 0.", got)
	}
	if got := m.Keys(); len(got) > 0 {
		t.Errorf("Got: map keys %v did not match kept keys. Want: [].", got)
	}

	// The only thing that a nil map can't safely handle is setting a key.
	func() {
		defer func() {
			recover()
		}()
		m.Set(i1, "abc")
		t.Errorf("Got: setting a new key on nil map did not panic, %s. Want: panic.", m.String())
	}()
}

func keysMatch(a, b []Instance) bool {
	if len(a) != len(b) {
		return false
	}
	found := make([]bool, len(b))
	for _, v := range a {
		i := keyAt(b, v)
		if i < 0 || found[i] {
			return false
		}
		found[i] = true
	}
	return true
}

func keyAt(keys []Instance, target Instance) int {
	for i, v := range keys {
		if v.Object == target.Object && v.TArgs.Equal(target.TArgs) {
			return i
		}
	}
	return -1
}
