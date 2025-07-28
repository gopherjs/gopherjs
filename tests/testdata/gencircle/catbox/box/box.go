package box

type Unboxer[T any] interface {
	Unbox() T
}

type boxImp[T any] struct {
	whatsInTheBox T
}

func Box[T any](value T) Unboxer[T] {
	return &boxImp[T]{whatsInTheBox: value}
}

func (b *boxImp[T]) Unbox() T {
	return b.whatsInTheBox
}
