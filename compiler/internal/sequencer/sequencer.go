package sequencer

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// ErrCycleDetected is panicked from a method performing sequencing
// (e.g. `Depth`, `DepthCount`, and `Group`) to indicate that a cycle
// was detected in the dependency graph.
var ErrCycleDetected = errors.New(`cycle detected in the dependency graph`)

// Sequencer is a tool for determining the groups and ordering of the groups
// of items based on their dependencies.
type Sequencer[T comparable] interface {

	// Dependency adds dependencies  items where the `child` is
	// dependent on the `parents`.
	Add(child T, parents ...T)

	// Has checks if an item exists in the sequencer.
	Has(item T) bool

	// Children returns the items that are dependent on the given item.
	// Each time this is called it creates a new slice.
	Children(item T) []T

	// Parents returns the items that given item depends on.
	// Each time this is called it creates a new slice.
	Parents(item T) []T

	// Depth returns the depth of the item in the dependency graph.
	// Zero indicates the item is a root item with no dependencies.
	// If this item doesn't exist -1 is returned.
	//
	// This may have to perform sequencing of the items, so
	// this may panic with `ErrCycleDetected` if a cycle is detected.
	Depth(item T) int

	// DepthCount returns the number of unique depths in the dependency graph.
	//
	// This may have to perform sequencing of the items, so
	// this may panic with `ErrCycleDetected` if a cycle is detected.
	DepthCount() int

	// Group returns all the items at the given depth.
	// If the depth is out of bounds, it returns an empty slice.
	// The depth is zero-based, so depth 0 is the root items.
	// Each time this is called it creates a new slice.
	//
	// This may have to perform sequencing of the items, so
	// this may panic with `ErrCycleDetected` if a cycle is detected.
	Group(depth int) []T

	// GetCycles returns the items that were unable to be sequenced
	// due to a cycle in the dependency graph.
	// The returned items may participate in one or more cycles or
	// depends on an item in a cycle.
	// Otherwise nil is returned if there are no cycles.
	//
	// These is no need to call this method before calling other methods.
	// If this returns a non-empty slice, other methods that perform sequencing
	// (e.g. `Depth`, `DepthCount`, and `Group`) will panic with `ErrCycleDetected`.
	//
	// This may have to perform sequencing of the items.
	GetCycles() []T

	// ToMermaid returns a string representation of the dependency graph in
	// Mermaid syntax. This is useful for visualizing the dependencies and
	// debugging dependency issues. When a cycle is detected, the items
	// will be marked with red and the groups may be incorrect.
	ToMermaid() string
}

// New creates a new sequencer for the given item type T.
func New[T comparable]() Sequencer[T] {
	return &sequencerImp[T]{
		vertices: vertexSet[T]{},
	}
}

type sequencerImp[T comparable] struct {
	// vertices is a set of all vertices indexed by the item they represent.
	vertices vertexSet[T]

	// needSequencing indicates that the sequencer needs to perform sequencing.
	needSequencing bool

	// depthCount is the number of unique depths in the dependency graph.
	// This is not valid if sequencing needs to be performed.
	depthCount int

	// groups is the map of groups indexed by their depth.
	// This may contain valid groups if sequencing needs to be performed.
	groups map[int]vertexSet[T]

	// dependencyCycles is the list of items that are part of any cycle
	// or depend on an item in a cycle.
	dependencyCycles vertexSet[T]
}

func (s *sequencerImp[T]) Add(child T, parents ...T) {
	c := s.getOrAdd(child)
	for _, parent := range parents {
		if !c.parents.has(parent) {
			p := s.getOrAdd(parent)
			c.addDependency(p)
		}
	}
}

func (s *sequencerImp[T]) Has(item T) bool {
	return s.vertices.has(item)
}

func (s *sequencerImp[T]) Children(item T) []T {
	if v, exists := s.vertices[item]; exists {
		return v.children.toSlice()
	}
	return nil
}

func (s *sequencerImp[T]) Parents(item T) []T {
	if v, exists := s.vertices[item]; exists {
		return v.parents.toSlice()
	}
	return nil
}

func (s *sequencerImp[T]) Depth(item T) int {
	s.performSequencing(true)
	if v, exists := s.vertices[item]; exists {
		return v.depth
	}
	return -1
}

func (s *sequencerImp[T]) DepthCount() int {
	s.performSequencing(true)
	return s.depthCount
}

func (s *sequencerImp[T]) Group(depth int) []T {
	s.performSequencing(true)
	return s.groups[depth].toSlice()
}

func (s *sequencerImp[T]) GetCycles() []T {
	s.performSequencing(false)
	return s.dependencyCycles.toSlice()
}

type sortByName[T comparable] struct {
	vertices []*vertex[T]
	names    []string
}

func (s *sortByName[T]) Len() int {
	return len(s.vertices)
}

func (s *sortByName[T]) Less(i, j int) bool {
	return s.names[i] < s.names[j]
}

func (s *sortByName[T]) Swap(i, j int) {
	s.vertices[i], s.vertices[j] = s.vertices[j], s.vertices[i]
	s.names[i], s.names[j] = s.names[j], s.names[i]
}

func (s *sequencerImp[T]) ToMermaid() string {
	s.performSequencing(false)

	buf := &bytes.Buffer{}
	write := func(format string, args ...any) {
		// Ignore the error since we are writing to a buffer.
		_, _ = buf.WriteString(fmt.Sprintf(format, args...))
	}

	// Sort the output to make it easier to read and compare consecutive runs.
	vertices := make([]*vertex[T], 0, len(s.vertices))
	names := make([]string, 0, len(vertices))
	for _, v := range s.vertices {
		vertices = append(vertices, v)
		names = append(names, fmt.Sprintf("%v", v.item))
	}
	sort.Sort(&sortByName[T]{vertices: vertices, names: names})

	ids := make(map[*vertex[T]]string, len(s.vertices))
	for i, v := range vertices {
		ids[v] = fmt.Sprintf(`v%d`, i)
	}

	toIds := func(vs vertexSet[T]) string {
		rs := make([]string, 0, len(vs))
		for _, v := range vs {
			rs = append(rs, ids[v])
		}
		sort.Strings(rs)
		return strings.Join(rs, ` & `)
	}

	write("flowchart TB\n")
	if len(s.dependencyCycles) > 0 {
		write("  classDef partOfCycle stroke:#f00\n")
	}
	for i, v := range vertices {
		write(`  %s["%v"]`, ids[v], names[i])
		if s.dependencyCycles.has(v.item) {
			write(`:::partOfCycle`)
		}
		if len(v.parents) > 0 {
			write(` --> %s`, toIds(v.parents))
		}
		write("\n")
	}
	for depth := 0; depth < s.depthCount; depth++ {
		if group := s.groups[depth]; len(group) > 0 {
			write("  subgraph Depth %d\n", depth)
			write("    %s\n", toIds(group))
			write("  end\n")
		}
	}
	return buf.String()
}

func (s *sequencerImp[T]) getOrAdd(item T) *vertex[T] {
	v, added := s.vertices.getOrAdd(item)
	s.needSequencing = s.needSequencing || added
	return v
}

// performSequencing performs a full sequencing of the items in the
// dependency graph. It calculates the depth of each item and groups
// them by their depth. If a cycle is detected, it panics.
//
// This assumes that the sequencing is not called often and is typically
// only called after all the items have been added. Because of this,
// it always performs a full sequencing of the items without using any
// previous solved information. Although this is slower for the few cases
// where sequencing happens often with only a few new items added at a time,
// it is much simpler to implement and maintain than implementing both
// incremental and full sequencing.
//
// `panicOnCycle“ indicates whether to panic if a cycle is detected,
// or to exit gracefully setting the `dependencyCycles` field.
func (s *sequencerImp[T]) performSequencing(panicOnCycle bool) {
	if !s.needSequencing {
		// If a sequencing was already performed and determined that there
		// was a cycle, panic if `panicOnCycle` is true.
		if len(s.dependencyCycles) > 0 && panicOnCycle {
			panic(ErrCycleDetected)
		}
		return
	}
	s.needSequencing = false

	// Perform a full sequencing of the items.
	s.clearGroups()
	ready := newVertexStack[T](len(s.vertices))
	waiting := s.prepareWaitingAndReady(true, s.vertices, ready)
	waiting = s.propagateDepth(true, waiting, ready)
	if waiting == 0 {
		return
	}

	// If there are still waiting vertices, it means there is a cycle.
	// Prune off any branches to leaves that are not part of the cycles
	// using the same logic except starting from the leaves.
	// This will not be able to remove branches that go between two cycles
	// even if vertices in that branch can not reach themselves via a cycle.
	wv := s.vertices.waitingVertices(waiting)
	waiting = s.prepareWaitingAndReady(false, wv, ready)
	waiting = s.propagateDepth(false, waiting, ready)
	s.dependencyCycles = make(vertexSet[T], waiting)
	for _, v := range s.vertices {
		if !v.isReady() {
			s.dependencyCycles.add(v)
		}
	}
	if panicOnCycle {
		panic(ErrCycleDetected)
	}
}

// clearGroups resets the sequencer state, clearing the groups and depth count.
// This is performed before performing a full sequencing since all those
// values will be recalculated.
func (s *sequencerImp[T]) clearGroups() {
	s.depthCount = 0
	s.groups = map[int]vertexSet[T]{}
	s.dependencyCycles = nil
}

// writeDepth updates the sequencer state with the depth of the given vertex.
func (s *sequencerImp[T]) writeDepth(v *vertex[T]) {
	depth := v.parents.maxDepth() + 1
	v.depth = depth
	if _, exists := s.groups[depth]; !exists {
		s.groups[depth] = vertexSet[T]{}
		if depth >= s.depthCount {
			s.depthCount = depth + 1
		}
	}
	s.groups[depth].add(v)
}

// prepareWaitingAndReady prepare the ready sets so that any root (or leaf) vertex
// is ready to be processed and any waiting vertex has its parent count.
// This returns the number of waiting vertices and a stack of ready vertices.
//
// If `forward` is true, it prepares the vertices for sequencing by starting with the roots.
// If `forward` is false, it prepares the vertices for reducing to cycles by starting with the leaves.
func (s *sequencerImp[T]) prepareWaitingAndReady(forward bool, vs vertexSet[T], ready *vertexStack[T]) int {
	waiting := 0
	for _, v := range vs {
		if forward {
			v.waiting = len(v.parents)
		} else {
			// For reducing to cycles, count the number of children that are still waiting.
			count := 0
			for _, c := range v.children {
				if vs.has(c.item) {
					count++
				}
			}
			v.waiting = count
		}

		if v.isReady() {
			s.writeDepth(v)
			ready.push(v)
		} else {
			waiting++
		}
	}
	return waiting
}

// propagateDepth processes the ready vertices, assigning them a depth and
// updating the waiting vertices. If a waiting vertex has all of its
// parents (or children) processed, then move it to the ready list.
// This continues until all ready vertices are processed.
func (s *sequencerImp[T]) propagateDepth(forward bool, waiting int, ready *vertexStack[T]) int {
	for ready.hasMore() {
		v := ready.pop()
		s.writeDepth(v)
		for _, c := range v.edges(forward) {
			c.decWaiting()
			if c.isReady() {
				ready.push(c)
				waiting--
			}
		}
	}
	return waiting
}
