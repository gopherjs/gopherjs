package dragon

type Burnable interface{ String() string }

type Trogdor[T Burnable] struct{}

func (t Trogdor[T]) Burninate(target T) {
	println("burninating the " + target.String())
}
