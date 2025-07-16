package sequencer

import "errors"

// ErrCycleDetected is panicked from a method performing sequencing
// (e.g. `Depth`, `DepthCount`, and `Group`) to indicate that a cycle
// was detected in the dependency graph.
var ErrCycleDetected = errors.New(`cycle detected in the dependency graph`)

// Sequencer is a tool for determining the groups and ordering of the groups
// of items based on their dependencies.
type Sequencer[T comparable] interface {

	// Add adds a `child` item with a dependency on the given `parents`.
	Add(child T, parents ...T)

	// Has checks if an item exists in the sequencer.
	Has(item T) bool

	// Children returns the items that are dependent on the given item.
	// If the given item doesn't exist then nil is returned.
	// Each time this is called it creates a new slice.
	// The items in the slice are in random order.
	Children(item T) []T

	// Parents returns the items that the given item depends on.
	// If the given item doesn't exist then nil is returned.
	// Each time this is called it creates a new slice.
	// The items in the slice are in random order.
	Parents(item T) []T

	// Depth returns the depth of the item in the dependency graph.
	// Zero indicates the item is a leaf item with no dependencies.
	// If the given item doesn't exist then -1 is returned.
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
	// If the depth is out-of-bounds, it returns an empty slice.
	// The depth is zero-based, so the depth 0 group is the leaf items.
	// Each time this is called it creates a new slice.
	// The items in the slice are in random order.
	//
	// This may have to perform sequencing of the items, so
	// this may panic with `ErrCycleDetected` if a cycle is detected.
	Group(depth int) []T

	// AllGroups returns all the items grouped by their depth.
	// Each group is a slice of items at the same depth.
	// The depth is zero-based, so the first group is the leaf items.
	// Each time this is called it creates a new slices.
	// The items in the slices are in random order.
	//
	// This may have to perform sequencing of the items, so
	// this may panic with `ErrCycleDetected` if a cycle is detected.
	AllGroups() [][]T

	// GetCycles returns the items that were unable to be sequenced
	// due to a cycle in the dependency graph.
	// The returned items may participate in one or more cycles or
	// depends on an item in a cycle.
	// Otherwise nil is returned if there are no cycles.
	//
	// There is no need to call this method before calling other methods.
	// If this returns a non-empty slice, other methods that perform sequencing
	// (e.g. `Depth`, `DepthCount`, and `Group`) will panic with `ErrCycleDetected`.
	// Obviously, this will not panic if a cycle is detected.
	//
	// This may have to perform sequencing of the items.
	GetCycles() []T

	// ToGraph returns a string representation of the dependency graph.
	// This is useful for visualizing the dependencies and
	// debugging dependency issues. When a cycle is detected, the items
	// participating in the cycle or depending on an item in a cycle
	// will be marked with red and the groups may be incorrect.
	ToGraph(options GraphOptions[T]) string
}

// GraphOptions contains options for the graph generation.
type GraphOptions[T comparable] struct {

	// ItemToString is used to convert the item to a string
	// representation for the graph nodes. It should return a unique string.
	// If nil, then `%v` will be used to convert the item to a string.
	ItemToString func(item T) string

	// FilterCycles indicates that the graph should only include
	// items that are part of a cycle or (if StrictFilter is false)
	// depend on an item in a cycle.
	// This will still run any item filter on the items in the cycle.
	FilterCycles bool

	// ItemFilter is used to reduce the items in the graph to only
	// those that match the filter and any of their children or parents.
	// If nil, then all items will be included in the graph.
	ItemFilter func(item T) bool

	// StrictFilter indicates that the graph should only include items
	// that match the filter and not their children or parents.
	StrictFilter bool

	// Mermaid indicates the graph should be output as a Mermaid graph
	// which can be used in Markdown files on Github but is limited in size.
	// Otherwise, the graph will be output as a DOT graph which can be used
	// with Graphviz and can handle larger graphs.
	Mermaid bool

	// HideGroups indicates that the groups should not be shown in the graph.
	// This is useful for large graphs where the groups are causing clutter.
	HideGroups bool

	// LabelItemsWithGroupNumber indicates that the items should be labeled
	// with their group number in the graph.
	LabelItemsWithGroupNumber bool
}

// New creates a new sequencer for the given item type T.
func New[T comparable]() Sequencer[T] {
	return &sequencerImp[T]{
		vertices: vertexSet[T]{},
	}
}
