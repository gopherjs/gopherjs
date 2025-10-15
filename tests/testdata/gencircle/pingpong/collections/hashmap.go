package collections

// HashSet keeps a set of non-nil elements that have unique hashes.
type HashSet[E Hashable] struct {
	data map[uint]E
}

func (s *HashSet[E]) Add(e E) {
	if s.data == nil {
		s.data = map[uint]E{}
	}
	s.data[e.Hash()] = e
}

func (s *HashSet[E]) Count() int {
	return len(s.data)
}
