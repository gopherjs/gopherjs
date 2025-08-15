package stable

import (
	"strings"

	"github.com/gopherjs/gopherjs/tests/testdata/gencircle/trammel/cmp"
	"github.com/gopherjs/gopherjs/tests/testdata/gencircle/trammel/collections"
	"github.com/gopherjs/gopherjs/tests/testdata/gencircle/trammel/sorts"
)

func MapString[K cmp.Ordered, V any, M ~map[K]V](m M, stringer func(K, V) string) string {
	// Function parameter with type parameters arguments.
	result := collections.KeysAndValues(m)
	keys := result.Keys
	values := result.Values

	sorts.Pair(keys, values)

	parts := zipper(keys, values, stringer)
	return `{` + strings.Join(parts, `, `) + `}`
}

func zipper[TIn1, TIn2, TOut any, SIn1 ~[]TIn1, SIn2 ~[]TIn2, F ~func(TIn1, TIn2) TOut](s1 SIn1, s2 SIn2, merge F) []TOut {
	// Overly complex type parameters including a generic function type.
	min := len(s1)
	if len(s2) < min {
		min = len(s2)
	}
	result := make([]any, min)
	for i := 0; i < min; i++ {
		result[i] = merge(s1[i], s2[i])
	}
	return castSlice[any, TOut](result)
}

func castSlice[TIn, TOut any, SIn ~[]TIn, SOut []TOut](s SIn) SOut {
	result := make(SOut, len(s))
	for i, v := range s {
		// Using a type parameter to cast the type.
		result[i] = any(v).(TOut)
	}
	return result
}
