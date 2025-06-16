package sequencer

// vertex represents a single item in the dependency graph.
type vertex[T comparable] struct {
	item     T
	depth    int
	parents  vertexSet[T]
	children vertexSet[T]
}

func newVertex[T comparable](item T) *vertex[T] {
	return &vertex[T]{
		item:  item,
		depth: -1,
	}
}

func (v *vertex[T]) addDependency(p *vertex[T]) {
	if p.children == nil {
		p.children = vertexSet[T]{}
	}
	p.children.add(v)

	if v.parents == nil {
		v.parents = vertexSet[T]{}
	}
	v.parents.add(p)
}
