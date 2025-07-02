package sequencer

type vertexStack[T comparable] struct {
	stack []*vertex[T]
}

func newVertexStack[T comparable](capacity int) *vertexStack[T] {
	return &vertexStack[T]{
		stack: make([]*vertex[T], 0, capacity),
	}
}

func (vs *vertexStack[T]) hasMore() bool {
	return len(vs.stack) > 0
}

func (vs *vertexStack[T]) push(v *vertex[T]) {
	vs.stack = append(vs.stack, v)
}

func (vs *vertexStack[T]) pop() *vertex[T] {
	maxIndex := len(vs.stack) - 1
	v := vs.stack[maxIndex]
	vs.stack = vs.stack[:maxIndex]
	return v
}
