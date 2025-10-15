package collections

type Hasher interface {
	Add(value uint) Hasher
	Sum() uint
}

type Hashable interface {
	Hash() uint
}

type BadHasher struct{ value uint }

func (h BadHasher) Add(value uint) Hasher {
	h.value += value
	return h
}

func (h BadHasher) Sum() uint {
	return h.value
}
