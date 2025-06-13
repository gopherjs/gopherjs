# Sequencer

- [Overview](#overview)
- [Limitations](#limitations)
- [Design](#design)
- [Ordering and Grouping](#ordering-and-grouping)

## Overview

The sequencer is a tool used to determine order the steps to process
a set of items based on each items' dependencies.
This can group items together if they can all be processed in the same step.
This assumes there are no circular dependencies and will error if any
cycle is detected.

The sequencer _could_ be used to solve several problems[^1], such as:

- Ordering type initialization
- Ordering constant resolution
- Ordering packages for parallel parsing
- Ordering computations for update propagation
- Dead code elimination (DCE)
- Explicit super-type determination to solidify implicit duck-typing

[^1]: We don't use the sequencer for all of those problems. Some are solved
with other tools and some are solutions we don't currently support. Several
of them would need an additional value tagging system added.

## Limitations

> [!IMPORTANT]
> The sequencer can only sequence a collection of items that don't have
> dependency cycles.

<!---->
> [!WARNING]
> The sequencer can only sequence items that are
> [comparable](https://go.dev/ref/spec#Comparison_operators)
>
> Since the items are used a keys in the graph, the comparable parts
> of an item must not be modified after being added,
> since that could cause keying issues.

The sequencer does not:

- sort items in the groups, each group is randomly ordered
- provide any weighted or prioritized dependencies
- provide a tagging system for data propagation needed for problems like DCE
- allow removing items from the graph nor removing dependencies

## Design

The sequencer uses a DAG (directed acyclic graph) where the vertices are the
items being ordered and the directed edge starts from a vertex, the parent, and
ends on a vertex, child, dependent of that parent.
This result is a tangled forest (a forest were branches may have more than
one parent branch).
Each vertex may have zero or more parents and zero or more children.
Any vertex that has no children (other vertices depending on it) is a leaf.
Any vertex that has no parents (dependencies) is a root.
The graph flows from the root towards the leaves via parent to child.

## Ordering and Grouping

All the root vertices will receive a depth of zero value.
All other nodes will receive the maximum value of its parents' depths plus one.
All the vertices with the same depth value are in a group and may be processed
together. The depth values provide the ordering of those groups.

To keep from having to recalculate a child's depth, each vertex will
keep a count of parents it is waiting on. When a vertex has its depth
assigned, that vertex's children will have that parent count decremented.
When that parent count is zero, the vertex will be put into the set of
vertices that need to be calculated.
If all the set of vertices pending calculation is empty and there are no
more vertices waiting on parents, then the depth determination is done.

However, if the set of vertices pending calculation is empty but there
are still vertices waiting on parents, then a cycle exists within those
vertices still waiting. Some of the vertices waiting my not participate
in the cycle but instead simply depend on vertices in the cycle.
There might also be multiple cycles. Cycles indicate the dependency
information given to the sequencer was bad so the sequencer will return
an error containing information about the cycle.

> [!NOTE]
> This assumes that the sequencing will be performed only once
> after all dependencies have been added. It doesn't have the partial
> sequencing capabilities, so it will always recalculate everything.
