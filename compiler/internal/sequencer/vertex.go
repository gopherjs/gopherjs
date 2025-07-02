package sequencer

// vertex represents a single item in the dependency graph.
type vertex[T comparable] struct {
	item     T
	depth    int
	parents  vertexSet[T]
	children vertexSet[T]

	// waiting is used during sequencing. Typically it contains the number of
	// parents that are not yet processed for this vertex.
	// When it reaches 0, the vertex can be processed.
	// It is used to avoid processing the same vertex multiple times.
	//
	// If a cycle is detected in the graph, this value can be used to
	// reduce the number of vertices that are dependent on the cycle.
	// In that case this value will be set to the number of children
	// that are not yet processed for this vertex.
	//
	// When reducing to cycles this can be negative for any vertex that was
	// ready during sequencing. We don't want those to be processed again,
	// so we only say a vertex is ready when waiting is zero.
	//
	// Since this number is only used during sequencing it could have been
	// stored in a map, however, it is faster to store it directly in the
	// vertex and avoid the map lookup.
	waiting int
}

func newVertex[T comparable](item T) *vertex[T] {
	return &vertex[T]{
		item:    item,
		depth:   -1,
		waiting: -1,
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

func (v *vertex[T]) edges(forward bool) vertexSet[T] {
	if forward {
		return v.children
	}
	return v.parents
}

func (v *vertex[T]) decWaiting() {
	v.waiting--
}

func (v *vertex[T]) isReady() bool {
	return v.waiting == 0
}
