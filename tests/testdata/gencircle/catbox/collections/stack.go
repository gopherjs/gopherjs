package collections

type Stack[T any] struct{ values []T }

func NewStack[T any]() *Stack[T] {
	return &Stack[T]{}
}

func (s *Stack[T]) Count() int {
	return len(s.values)
}

func (s *Stack[T]) Push(value T) {
	s.values = append(s.values, value)
}

func (s *Stack[T]) Pop() (value T) {
	if len(s.values) > 0 {
		maxIndex := len(s.values) - 1
		s.values, value = s.values[:maxIndex], s.values[maxIndex]
	}
	return
}
