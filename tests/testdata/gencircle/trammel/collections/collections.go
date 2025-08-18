package collections

func Populate[K comparable, V any, SK ~[]K, SV ~[]V, M ~map[K]V](m M, keys SK, values SV) {
	// Lots of type parameters with parameters referencing each other.
	for i, k := range keys {
		if i < len(values) {
			m[k] = values[i]
		} else {
			var zero V
			m[k] = zero
		}
	}
}

func KeysAndValues[K comparable, V any, M ~map[K]V](m M) struct {
	Keys   []K
	Values []V
} {
	keys := make([]K, 0, len(m))
	values := make([]V, 0, len(m))
	for k, v := range m {
		keys = append(keys, k)
		values = append(values, v)
	}
	// nested generic type that has a type parameter and nest type parameter.
	type result[T any] struct {
		Keys   []T
		Values []V
	}
	return result[K]{Keys: keys, Values: values}
}
