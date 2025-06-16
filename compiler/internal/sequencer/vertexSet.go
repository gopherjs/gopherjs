package sequencer

// vertexSet is a set of vertices indexed by the item the represent.
//
// The values will be unique since vertices contain the item themselves,
// and no two vertices can represent the same item in the graph,
// meaning this map is bijective.
type vertexSet[T comparable] map[T]*vertex[T]

func (vs vertexSet[T]) add(v *vertex[T]) {
	vs[v.item] = v
}

func (vs vertexSet[T]) getOrAdd(item T) (*vertex[T], bool) {
	if v, exists := vs[item]; exists {
		return v, false
	}

	v := newVertex(item)
	vs[item] = v
	return v, true
}

func (vs vertexSet[T]) has(item T) bool {
	_, exists := vs[item]
	return exists
}

func (vs vertexSet[T]) maxDepth() int {
	maxDepth := -1
	for _, v := range vs {
		if v.depth > maxDepth {
			maxDepth = v.depth
		}
	}
	return maxDepth
}

func (vs vertexSet[T]) toSlice() []T {
	items := make([]T, 0, len(vs))
	for item := range vs {
		items = append(items, item)
	}
	return items
}
