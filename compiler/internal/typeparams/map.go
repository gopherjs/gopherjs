package typeparams

import (
	"fmt"
	"go/types"
	"sort"
	"strings"

	"golang.org/x/tools/go/types/typeutil"
)

type (
	mapEntry[V any] struct {
		key   Instance
		value V
	}
	mapBucket[V any]  []*mapEntry[V]
	mapBuckets[V any] map[uint32]mapBucket[V]
)

// InstanceMap implements a map-like data structure keyed by instances.
//
// Zero value is an equivalent of an empty map. Methods are not thread-safe.
//
// Since Instance contains a slice and is not comparable, it can not be used as
// a regular map key, but we can compare its fields manually. When comparing
// instance equality, objects are compared by pointer equality, and type
// arguments with types.Identical(). To reduce access complexity, we bucket
// entries by a combined hash of type args. This type is generally inspired by
// golang.org/x/tools/go/types/typeutil/map.go
type InstanceMap[V any] struct {
	data   map[types.Object]mapBuckets[V]
	len    int
	hasher typeutil.Hasher
}

// getBucket will return the bucket for the given key
// or an empty bucket if no bucket for that key exists.
func (im *InstanceMap[V]) getBucket(key Instance) mapBucket[V] {
	if im == nil || im.data == nil {
		return nil
	}
	return im.data[key.Object][typeHash(im.hasher, key.TArgs...)]
}

// get returns the stored value for the provided key and
// a bool indicating whether the key was present in the map or not.
func (im *InstanceMap[V]) get(key Instance) (V, bool) {
	bucket := im.getBucket(key)
	for _, candidate := range bucket {
		if candidate != nil && typeArgsEq(candidate.key.TArgs, key.TArgs) {
			return candidate.value, true
		}
	}
	var zero V
	return zero, false
}

// Get returns the stored value for the provided key. If the key is missing from
// the map, zero value is returned.
func (im *InstanceMap[V]) Get(key Instance) V {
	val, _ := im.get(key)
	return val
}

// Has returns true if the given key is present in the map.
func (im *InstanceMap[V]) Has(key Instance) bool {
	_, ok := im.get(key)
	return ok
}

// Set new value for the key in the map. Returns the previous value that was
// stored in the map, or zero value if the key wasn't present before.
func (im *InstanceMap[V]) Set(key Instance, value V) V {
	if im.data == nil {
		im.data = map[types.Object]mapBuckets[V]{}
		im.hasher = typeutil.MakeHasher()
	}

	if _, ok := im.data[key.Object]; !ok {
		im.data[key.Object] = mapBuckets[V]{}
	}
	bucketID := typeHash(im.hasher, key.TArgs...)

	// If there is already an identical key in the map, override the entry value.
	hole := -1
	bucket := im.data[key.Object][bucketID]
	for i, candidate := range bucket {
		if candidate == nil {
			hole = i
		} else if typeArgsEq(candidate.key.TArgs, key.TArgs) {
			old := candidate.value
			candidate.value = value
			return old
		}
	}

	// If there is a hole in the bucket, reuse it.
	if hole >= 0 {
		im.data[key.Object][bucketID][hole] = &mapEntry[V]{
			key:   key,
			value: value,
		}
	} else {
		// Otherwise append a new entry.
		im.data[key.Object][bucketID] = append(bucket, &mapEntry[V]{
			key:   key,
			value: value,
		})
	}
	im.len++
	var zero V
	return zero
}

// Len returns the number of elements in the map.
func (im *InstanceMap[V]) Len() int {
	if im != nil {
		return im.len
	}
	return 0
}

// Delete removes the entry with the given key, if any.
// It returns true if the entry was found.
func (im *InstanceMap[V]) Delete(key Instance) bool {
	bucket := im.getBucket(key)
	for i, candidate := range bucket {
		if candidate != nil && typeArgsEq(candidate.key.TArgs, key.TArgs) {
			// We can't compact the bucket as it
			// would disturb iterators.
			bucket[i] = nil
			im.len--
			return true
		}
	}
	return false
}

// Iterate calls function f on each entry in the map in unspecified order.
//
// Return true from f to continue the iteration, or false to stop it.
//
// If f should mutate the map, Iterate provides the same guarantees as
// Go maps: if f deletes a map entry that Iterate has not yet reached,
// f will not be invoked for it, but if f inserts a map entry that
// Iterate has not yet reached, whether or not f will be invoked for
// it is unspecified.
func (im *InstanceMap[V]) Iterate(f func(key Instance, value V) bool) {
	if im != nil && im.data != nil {
		for _, mapBucket := range im.data {
			for _, bucket := range mapBucket {
				for _, e := range bucket {
					if e != nil && !f(e.key, e.value) {
						return
					}
				}
			}
		}
	}
}

// Keys returns a new slice containing the set of map keys.
// The order is unspecified.
func (im *InstanceMap[V]) Keys() []Instance {
	keys := make([]Instance, 0, im.Len())
	im.Iterate(func(key Instance, _ V) bool {
		keys = append(keys, key)
		return true
	})
	return keys
}

// String returns a string representation of the map's entries.
// The entries are sorted by string representation of the entry.
func (im *InstanceMap[V]) String() string {
	entries := make([]string, 0, im.Len())
	im.Iterate(func(key Instance, value V) bool {
		entries = append(entries, fmt.Sprintf("%v:%v", key, value))
		return true
	})
	sort.Strings(entries)
	return `{` + strings.Join(entries, `, `) + `}`
}

// typeHash returns a combined hash of several types.
//
// Provided hasher is used to compute hashes of individual types, which are
// xor'ed together. Xor preserves bit distribution property, so the combined
// hash should be as good for bucketing, as the original.
func typeHash(hasher typeutil.Hasher, types ...types.Type) uint32 {
	var hash uint32
	for _, typ := range types {
		hash ^= hasher.Hash(typ)
	}
	return hash
}

// typeArgsEq returns if both lists of type arguments are identical.
func typeArgsEq(a, b []types.Type) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !types.Identical(a[i], b[i]) {
			return false
		}
	}

	return true
}
