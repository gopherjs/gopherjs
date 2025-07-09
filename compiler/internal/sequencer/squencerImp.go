package sequencer

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// errSequencerLogic is panicked if an error internal to the sequencer logic
// or error in the parent/child pointers is detected.
// This error should never be panicked if the sequencer is working correctly.
var errSequencerLogic = errors.New(`error in sequencer logic or parent/child pointers`)

type sequencerImp[T comparable] struct {
	// vertices is a set of all vertices indexed by the item they represent.
	vertices vertexSet[T]

	// needSequencing indicates that the sequencer needs to perform sequencing.
	needSequencing bool

	// depthCount is the number of unique depths in the dependency graph.
	// This may be invalid if sequencing needs to be performed.
	depthCount int

	// groups is the map of groups indexed by their depth.
	// This may contain invalid groups if sequencing needs to be performed.
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

func (s *sequencerImp[T]) AllGroups() [][]T {
	s.performSequencing(true)
	groups := make([][]T, s.depthCount)
	for depth := 0; depth < s.depthCount; depth++ {
		groups[depth] = s.groups[depth].toSlice()
	}
	return groups
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

func (s *sequencerImp[T]) ToMermaid(itemToString func(item T) string) string {
	s.performSequencing(false)

	if itemToString == nil {
		itemToString = func(item T) string {
			return fmt.Sprintf("%v", item)
		}
	}

	buf := &bytes.Buffer{}
	write := func(format string, args ...any) {
		// Ignore the error since we are writing to a buffer.
		_, _ = fmt.Fprintf(buf, format, args...)
	}

	// Sort the output to make it easier to read and compare consecutive runs.
	vertices := make([]*vertex[T], 0, len(s.vertices))
	names := make([]string, 0, len(vertices))
	for _, v := range s.vertices {
		vertices = append(vertices, v)
		names = append(names, itemToString(v.item))
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
	for depth := s.depthCount - 1; depth >= 0; depth-- {
		if group := s.groups[depth]; len(group) > 0 {
			write("  subgraph Depth %d\n", depth)
			write("    %s\n", toIds(group))
			write("  end\n")
		}
	}
	return buf.String()
}

func (s *sequencerImp[T]) ToDot(itemToString func(item T) string) string {
	s.performSequencing(false)

	if itemToString == nil {
		itemToString = func(item T) string {
			return fmt.Sprintf("%v", item)
		}
	}

	buf := &bytes.Buffer{}
	write := func(format string, args ...any) {
		// Ignore the error since we are writing to a buffer.
		_, _ = fmt.Fprintf(buf, format, args...)
	}

	// Sort the output to make it easier to read and compare consecutive runs.
	vertices := make([]*vertex[T], 0, len(s.vertices))
	names := make([]string, 0, len(vertices))
	for _, v := range s.vertices {
		vertices = append(vertices, v)
		names = append(names, itemToString(v.item))
	}
	sort.Sort(&sortByName[T]{vertices: vertices, names: names})

	ids := make(map[*vertex[T]]string, len(s.vertices))
	for i, v := range vertices {
		ids[v] = fmt.Sprintf(`v%d`, i)
	}

	toIds := func(vs vertexSet[T]) []string {
		rs := make([]string, 0, len(vs))
		for _, v := range vs {
			rs = append(rs, ids[v])
		}
		sort.Strings(rs)
		return rs
	}

	write("digraph G {\n")
	for i, v := range vertices {
		write("\t%s[label=%q", ids[v], names[i])
		if s.dependencyCycles.has(v.item) {
			write(`,color=red`)
		}
		write("]\n")
	}
	for _, v := range vertices {
		if len(v.parents) > 0 {
			write("\t%s -> {%s}\n", ids[v], strings.Join(toIds(v.parents), ` `))
		}
	}
	for depth := s.depthCount - 1; depth >= 0; depth-- {
		if group := s.groups[depth]; len(group) > 0 {
			write("\tsubgraph depth_%d {\n", depth)
			write("\t\tlabel = \"Depth %d\"\n", depth)
			write("\t\t%s;\n", strings.Join(toIds(group), `; `))
			write("\t}\n")
		}
	}
	write("}\n")
	return buf.String()
}

func (s *sequencerImp[T]) getOrAdd(item T) *vertex[T] {
	v, added := s.vertices.getOrAdd(item)
	s.needSequencing = s.needSequencing || added
	return v
}

// performSequencing performs a full sequencing of the items in the
// dependency graph. It calculates the depth of each item and groups
// them by their depth.
//
// `panicOnCycle“ indicates whether to panic if a cycle is detected,
// or to exit gracefully.
//
// This assumes that the sequencing is not called often and is typically
// only called after all the items have been added. Because of this,
// it always performs a full sequencing of the items without using any
// previous solved information. Although this is slower for the few cases
// where sequencing happens often with only a few new items added at a time,
// it is much simpler to implement and maintain full sequencing than
// implementing both incremental and full sequencing.
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
	waitingCount := s.prepareWaitingAndReady(true, s.vertices, ready)
	waitingCount = s.propagateDepth(true, waitingCount, ready)
	if waitingCount <= 0 {
		s.validateGroups()
		return
	}

	// If there are still waiting vertices, it means there is a cycle.
	// Prune off any branches to roots that are not part of the cycles
	// using the same logic that starts from the leaves except starting
	// from the roots and working backwards.
	// This will not be able to remove branches that go between two cycles
	// even if vertices in that branch can not reach themselves via a cycle.
	wv := s.vertices.getWaiting(waitingCount)
	waitingCount = s.prepareWaitingAndReady(false, wv, ready)
	waitingCount = s.propagateDepth(false, waitingCount, ready)

	// Sanity check that we have waiting vertices left and we didn't
	// somehow prune away the vertices participating in the cycles.
	if waitingCount <= 0 {
		panic(fmt.Errorf(`%w: pruning cycles resulting in no items in the cycles`, errSequencerLogic))
	}

	// Anything still waiting is part of a cycle or depends on an item in a
	// cycle that wasn't able to be pruned.
	s.dependencyCycles = s.vertices.getWaiting(waitingCount)
	if panicOnCycle {
		panic(ErrCycleDetected)
	}
}

// clearGroups resets the sequencer state, clearing the groups and depth count.
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

// prepareWaitingAndReady prepare the ready sets so that any leaf (or root) vertex
// is ready to be processed and any waiting vertex has its parent count.
// This returns the number of waiting vertices.
//
// If `forward` is true, it prepares the vertices for sequencing by starting with the leaves.
// If `forward` is false, it prepares the vertices for reducing to cycles by starting with the roots.
func (s *sequencerImp[T]) prepareWaitingAndReady(forward bool, vs vertexSet[T], ready *vertexStack[T]) int {
	waitingCount := 0
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
			waitingCount++
		}
	}
	return waitingCount
}

// propagateDepth processes the ready vertices, assigning them a depth and
// updating the waiting vertices. If a waiting vertex has all of its
// parents (or children) processed, then move it to the ready list.
// This continues until all ready vertices are processed.
func (s *sequencerImp[T]) propagateDepth(forward bool, waitingCount int, ready *vertexStack[T]) int {
	for ready.hasMore() {
		v := ready.pop()
		s.writeDepth(v)
		for _, c := range v.edges(forward) {
			c.decWaiting()
			if c.isReady() {
				ready.push(c)
				waitingCount--
			}
		}
	}
	return waitingCount
}

// validateGroups validates that the groups and depths are correctly formed.
// This is a sanity check to ensure that the sequencer logic appears correct.
func (s *sequencerImp[T]) validateGroups() {
	if s.depthCount <= 0 {
		panic(fmt.Errorf(`%w: depth count is invalid`, errSequencerLogic))
	}
	count := 0
	for depth := 0; depth < s.depthCount; depth++ {
		group := s.groups[depth]
		if len(group) == 0 {
			panic(fmt.Errorf(`%w: group %d is empty`, errSequencerLogic, depth))
		}
		for _, v := range group {
			if v.depth != depth {
				panic(fmt.Errorf(`%w: vertex %v in group %d has depth %d`, errSequencerLogic, v.item, depth, v.depth))
			}
			hasPrior := false
			for _, p := range v.parents {
				if p.depth >= v.depth {
					panic(fmt.Errorf(`%w: vertex %v has parent %v with depth %d that is not less than its depth %d`, errSequencerLogic, v.item, p.item, p.depth, v.depth))
				}
				hasPrior = hasPrior || p.depth == v.depth-1
			}
			if depth > 0 && !hasPrior {
				panic(fmt.Errorf(`%w: vertex %v in group %d has no parent with depth %d`, errSequencerLogic, v.item, depth, v.depth-1))
			}
		}
		count += len(group)
	}
	if count != len(s.vertices) {
		panic(fmt.Errorf(`%w: vertices in groups, %d, does not match vertex count %d`, errSequencerLogic, count, len(s.vertices)))
	}
}
