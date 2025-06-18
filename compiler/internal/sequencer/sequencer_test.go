package sequencer

import (
	"errors"
	"math/rand"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestBasicSequencing(t *testing.T) {
	s := New[string]()
	s.Add(`Rad`, `Bob`, `Chris`)
	s.Add(`Stripe`, `Bob`, `Chris`)
	s.Add(`Bandit`, `Bob`, `Chris`)
	s.Add(`Brandy`, `Mort`)
	s.Add(`Chili`, `Mort`)
	s.Add(`Muffin`, `Stripe`, `Trixie`)
	s.Add(`Socks`, `Stripe`, `Trixie`)
	s.Add(`Bluey`, `Bandit`, `Chili`)
	s.Add(`Bingo`, `Bandit`, `Chili`)
	s.Add(`Frisky`)

	if !s.Has(`Bob`) {
		t.Errorf(`expected to find Bob in sequencer, but did not`)
	}
	if s.Has(`Ted`) {
		t.Errorf(`expected to not find Ted in sequencer, but did not`)
	}

	gotC := s.Children(`Bandit`)
	sort.Strings(gotC)
	expC := []string{`Bingo`, `Bluey`}
	if diff := cmp.Diff(gotC, expC); len(diff) > 0 {
		t.Errorf("unexpected children (-got +exp):\n%s", diff)
	}
	if gotC := s.Children(`Ted`); len(gotC) != 0 {
		t.Errorf("expected no children for an item not in the sequencer, got: %v", gotC)
	}

	gotP := s.Parents(`Bandit`)
	sort.Strings(gotP)
	expP := []string{`Bob`, `Chris`}
	if diff := cmp.Diff(gotP, expP); len(diff) > 0 {
		t.Errorf("unexpected parents (-got +exp):\n%s", diff)
	}
	if gotP := s.Parents(`Ted`); len(gotP) != 0 {
		t.Errorf("expected no parents for an item not in the sequencer, got: %v", gotP)
	}

	if depth := s.Depth(`Bandit`); depth != 1 {
		t.Errorf("expected depth of Bandit to be 1, got: %d", depth)
	}
	if depth := s.Depth(`Ted`); depth != -1 {
		t.Errorf("expected depth of an item not in the sequencer to be -1, got: %d", depth)
	}

	t.Log(s.ToMermaid())

	// Check getting the groups individually.
	count := s.DepthCount()
	got := make([][]string, count)
	for i := 0; i < s.DepthCount(); i++ {
		group := s.Group(i)
		sort.Strings(group)
		got[i] = group
	}
	exp := [][]string{
		{`Bob`, `Chris`, `Frisky`, `Mort`, `Trixie`},
		{`Bandit`, `Brandy`, `Chili`, `Rad`, `Stripe`},
		{`Bingo`, `Bluey`, `Muffin`, `Socks`},
	}
	if diff := cmp.Diff(got, exp); len(diff) > 0 {
		t.Errorf("unexpected sequencing (-got +exp):\n%s", diff)
	}

	// Using AllGroups should return the same result as reading the groups individually.
	got = s.AllGroups()
	for _, group := range got {
		sort.Strings(group)
	}
	if diff := cmp.Diff(got, exp); len(diff) > 0 {
		t.Errorf("unexpected sequencing (-got +exp):\n%s", diff)
	}
}

func TestDiamonds(t *testing.T) {
	s := New[string]()
	// This makes several diamonds in the graph to check that vertices
	// are only processed once all the parents are processed.
	s.Add(`A`, `B`, `C`, `D`, `G`)
	s.Add(`B`, `D`, `E`)
	s.Add(`C`, `D`, `F`)
	s.Add(`D`, `G`)
	s.Add(`E`, `G`)
	s.Add(`F`, `G`)

	t.Log(s.ToMermaid())

	got := s.AllGroups()
	for _, group := range got {
		sort.Strings(group)
	}
	exp := [][]string{
		{`G`},
		{`D`, `E`, `F`},
		{`B`, `C`},
		{`A`},
	}
	if diff := cmp.Diff(got, exp); len(diff) > 0 {
		t.Errorf("unexpected sequencing (-got +exp):\n%s", diff)
	}
}

func TestCycleDetection(t *testing.T) {
	s := New[string]()
	s.Add(`A`, `B`, `D`) // D is a root not part of the cycle
	s.Add(`B`, `C`, `D`)
	s.Add(`C`, `A`) // This creates a cycle A-> B->C->A
	s.Add(`E`, `A`) // E is a branch not part of the cycle

	t.Log(s.ToMermaid()) // Should not panic

	// Add more to reset the sequencer state
	s.Add(`F`, `E`) // F is a leaf via E not part of the cycle

	expectPanic := func(h func()) {
		defer func() {
			r := recover().(error)
			if !errors.Is(r, ErrCycleDetected) {
				t.Errorf(`expected panic due to cycle, but got: %v`, r)
			}
		}()
		h()
		s.DepthCount()
		t.Errorf(`expected panic due to cycle, but did not panic`)
	}

	expectPanic(func() { s.DepthCount() })
	expectPanic(func() { s.Depth(`A`) })
	expectPanic(func() { s.Group(2) })

	t.Log(s.ToMermaid()) // Should not panic

	cycles := s.GetCycles()
	sort.Strings(cycles)
	exp := []string{`A`, `B`, `C`}
	if diff := cmp.Diff(cycles, exp); len(diff) > 0 {
		t.Errorf("unexpected cycles (-got +exp):\n%s", diff)
	}
}

func TestLargeGraph(t *testing.T) {
	const itemCount = 1000
	const maxDeps = 10

	items := make([]int, itemCount)
	for i := 0; i < itemCount; i++ {
		items[i] = i
	}

	r := rand.New(rand.NewSource(0))
	r.Shuffle(itemCount, func(i, j int) {
		items[i], items[j] = items[j], items[i]
	})

	s := New[int]()
	for i := 0; i < maxDeps; i++ {
		s.Add(items[i]) // Add root items with no dependencies
	}
	for i := maxDeps; i < itemCount; i++ {
		s.Add(items[i])

		// "Randomly" add dependencies to previous items, since only previous
		// items are chosen from no cycles should occur.
		// If the same item is chosen multiple times it should have no effect.
		depCount := r.Intn(maxDeps)
		for j := 0; j < depCount; j++ {
			s.Add(items[i], items[r.Intn(i)])
		}
	}

	s.DepthCount() // This should not panic and internal validation should pass.
	if len(s.GetCycles()) > 0 {
		t.Errorf(`expected no cycles in the large graph, but found some`)
	}
}
