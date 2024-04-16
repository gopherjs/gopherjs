package typeparams

import (
	"go/types"
	"sync"

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
// typeutil.Map.
type InstanceMap[V any] struct {
	bootstrap sync.Once
	data      map[types.Object]mapBuckets[V]
	len       int
	hasher    typeutil.Hasher
	zero      V
}

func (im *InstanceMap[V]) init() {
	im.bootstrap.Do(func() {
		im.data = map[types.Object]mapBuckets[V]{}
		im.hasher = typeutil.MakeHasher()
	})
}

func (im *InstanceMap[V]) get(key Instance) (V, bool) {
	im.init()

	buckets, ok := im.data[key.Object]
	if !ok {
		return im.zero, false
	}
	bucket := buckets[typeHash(im.hasher, key.TArgs...)]
	if len(bucket) == 0 {
		return im.zero, false
	}

	for _, candidate := range bucket {
		if typeArgsEq(candidate.key.TArgs, key.TArgs) {
			return candidate.value, true
		}
	}
	return im.zero, false
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
func (im *InstanceMap[V]) Set(key Instance, value V) (old V) {
	im.init()

	if _, ok := im.data[key.Object]; !ok {
		im.data[key.Object] = mapBuckets[V]{}
	}
	bucketID := typeHash(im.hasher, key.TArgs...)

	// If there is already an identical key in the map, override the entry value.
	for _, candidate := range im.data[key.Object][bucketID] {
		if typeArgsEq(candidate.key.TArgs, key.TArgs) {
			old = candidate.value
			candidate.value = value
			return old
		}
	}

	// Otherwise append a new entry.
	im.data[key.Object][bucketID] = append(im.data[key.Object][bucketID], &mapEntry[V]{
		key:   key,
		value: value,
	})
	im.len++
	return im.zero
}

// Len returns the number of elements in the map.
func (im *InstanceMap[V]) Len() int {
	return im.len
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
