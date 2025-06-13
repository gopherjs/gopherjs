package sequencer

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
)

// Sequencer is a tool for determining the groups and ordering of the groups
// of items based on their dependencies.
type Sequencer[T comparable] interface {

	// Dependency adds dependencies  items where the `child` is
	// dependent on the `parents`.
	Add(child T, parents ...T)

	// Has checks if an item exists in the sequencer.
	Has(item T) bool

	// Dependents returns the items that are dependent on the given item.
	// Each time this is called it creates a new slice.
	Dependents(item T) []T

	// Dependencies returns the items that given item depends on.
	// Each time this is called it creates a new slice.
	Dependencies(item T) []T

	// Depth returns the depth of the item in the dependency graph.
	// Zero indicates the item is a root item with no dependencies.
	// If this item doesn't exist -1 is returned.
	//
	// This may have to perform sequencing of the items, so
	// this may panic if a cycle is detected.
	Depth(item T) int

	// DepthCount returns the number of unique depths in the dependency graph.
	//
	// This may have to perform sequencing of the items, so
	// this may panic if a cycle is detected.
	DepthCount() int

	// Group returns all the items at the given depth.
	// If the depth is out of bounds, it returns an empty slice.
	// The depth is zero-based, so depth 0 is the root items.
	// Each time this is called it creates a new slice.
	//
	// This may have to perform sequencing of the items, so
	// this may panic if a cycle is detected.
	Group(depth int) []T

	// ToMermaid returns a string representation of the dependency graph in
	// Mermaid syntax. This is useful for visualizing the dependencies and
	// debugging dependency issues.
	ToMermaid() string
}

// New creates a new sequencer for the given item type T.
func New[T comparable]() Sequencer[T] {
	return &sequencerImp[T]{
		vertices: vertexSet[T]{},
		groups:   map[int]vertexSet[T]{},
	}
}

var ErrCycleDetected = errors.New(`cycle detected in the dependency graph`)

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
}

func (s *sequencerImp[T]) Add(child T, parents ...T) {
	c := s.getOrAdd(child)
	for _, parent := range parents {
		if !c.hasParent(parent) {
			p := s.getOrAdd(parent)
			c.addDependency(p)
		}
	}
}

func (s *sequencerImp[T]) Has(item T) bool {
	return s.vertices.has(item)
}

func (s *sequencerImp[T]) Dependents(item T) []T {
	if v, exists := s.vertices[item]; exists {
		return v.children.toSlice()
	}
	return nil
}

func (s *sequencerImp[T]) Dependencies(item T) []T {
	if v, exists := s.vertices[item]; exists {
		return v.parents.toSlice()
	}
	return nil
}

func (s *sequencerImp[T]) Depth(item T) int {
	s.performSequencing()
	if v, exists := s.vertices[item]; exists {
		return v.depth
	}
	return -1
}

func (s *sequencerImp[T]) DepthCount() int {
	s.performSequencing()
	return s.depthCount
}

func (s *sequencerImp[T]) Group(depth int) []T {
	s.performSequencing()
	return s.groups[depth].toSlice()
}

func (s *sequencerImp[T]) ToMermaid() string {
	// Try to perform sequencing but ignore if it panics.
	func() {
		defer func() {
			_ = recover()
		}()
		s.performSequencing()
	}()

	buf := &bytes.Buffer{}
	write := func(format string, args ...any) {
		if _, err := buf.WriteString(fmt.Sprintf(format, args...)); err != nil {
			panic(fmt.Errorf(`failed to write to buffer: %w`, err))
		}
	}
	ids := make(map[*vertex[T]]string, len(s.vertices))
	getId := func(v *vertex[T]) string {
		if id, exists := ids[v]; exists {
			return id
		}
		id := fmt.Sprintf(`v%d`, len(ids)+1)
		ids[v] = id
		return id
	}
	toIds := func(vs vertexSet[T]) string {
		ids := make([]string, 0, len(vs))
		for _, v := range vs {
			ids = append(ids, getId(v))
		}
		return strings.Join(ids, ` & `)
	}

	write("flowchart TB\n")
	for _, v := range s.vertices {
		write(`  %s["%v"]`, getId(v), v.item)
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
func (s *sequencerImp[T]) performSequencing() {
	if !s.needSequencing {
		return
	}

	s.clearGroups()
	waiting, ready := s.prepareWaitingAndReady()
	s.propagateDepth(waiting, ready)
	s.checkForCycles(waiting)
	s.writeGroups()
	s.needSequencing = false
}

func (s *sequencerImp[T]) clearGroups() {
	s.depthCount = 0
	s.groups = map[int]vertexSet[T]{}
}

// prepareWaitingAndReady prepare the waiting and ready sets so that any root vertex
// is ready to be processed and any waiting vertex has its parent count.
func (s *sequencerImp[T]) prepareWaitingAndReady() (waiting map[*vertex[T]]int, ready []*vertex[T]) {
	waiting = make(map[*vertex[T]]int, len(s.vertices))
	ready = make([]*vertex[T], 0, len(s.vertices))
	for _, v := range s.vertices {
		parentCount := len(v.parents)
		if parentCount <= 0 {
			v.depth = 0
			ready = append(ready, v)
		} else {
			waiting[v] = parentCount
		}
	}
	return waiting, ready
}

// propagateDepth processes the ready vertices, assigning them a depth and
// updating the waiting vertices. If a waiting vertex has all its
// parents processed, move it to the ready list.
func (s *sequencerImp[T]) propagateDepth(waiting map[*vertex[T]]int, ready []*vertex[T]) {
	for len(ready) > 0 {
		maxIndex := len(ready) - 1
		v := ready[maxIndex]
		ready = ready[:maxIndex]

		v.depth = v.maxParentDepth() + 1
		for _, c := range v.children {
			waiting[c]--
			if waiting[c] <= 0 {
				ready = append(ready, c)
				delete(waiting, c)
			}
		}
	}
}

// checkForCycles checks if there are still waiting vertices. If there are
// it means there is a cycle since some of them are waiting for parents that
// eventually depend on them.
func (s *sequencerImp[T]) checkForCycles(waiting map[*vertex[T]]int) {
	if len(waiting) > 0 {
		// TODO: Add more information about the cycle.
		panic(ErrCycleDetected)
	}
}

// writeGroups update the sequencer state with the new depths and groups.
func (s *sequencerImp[T]) writeGroups() {
	for _, v := range s.vertices {
		depth := v.depth
		if depth < 0 {
			panic(fmt.Errorf(`vertex %v has no depth assigned`, v.item))
		}
		if _, exists := s.groups[depth]; !exists {
			s.groups[depth] = vertexSet[T]{}
		}
		s.groups[depth][v.item] = v
	}

	s.depthCount = len(s.groups)

	// Validate the groups to ensure they are correctly formed.
	for depth, group := range s.groups {
		if depth < 0 && depth >= s.depthCount {
			panic(fmt.Errorf(`depth %d is out of bounds, depth count is %d`, depth, s.depthCount))
		}
		if len(group) == 0 {
			panic(fmt.Errorf(`group at depth %d is empty`, depth))
		}
	}
}
