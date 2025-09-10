package otherpkg

var Test float32

func Zero[T any]() T {
	var zero T
	return zero
}

type GetterHandle[T any] func() T
