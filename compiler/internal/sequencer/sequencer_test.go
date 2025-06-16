package sequencer

import (
	"errors"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSequencingStrings(t *testing.T) {
	s := New[string]()

	s.Add(`Bob`, `Rad`, `Stripe`, `Bandit`)
	s.Add(`Chris`, `Rad`, `Stripe`, `Bandit`)
	s.Add(`Stripe`, `Muffin`, `Socks`)
	s.Add(`Trixie`, `Muffin`, `Socks`)
	s.Add(`Mort`, `Brandy`, `Chili`)
	s.Add(`Bandit`, `Bluey`, `Bingo`)
	s.Add(`Chili`, `Bluey`, `Bingo`)
	s.Add(`Frisky`)

	count := s.DepthCount()
	got := make([][]string, count)
	for i := 0; i < s.DepthCount(); i++ {
		group := s.Group(i)
		sort.Strings(group)
		got[i] = group
	}

	t.Log(s.ToMermaid())

	exp := [][]string{
		{`Bingo`, `Bluey`, `Brandy`, `Frisky`, `Muffin`, `Rad`, `Socks`},
		{`Bandit`, `Chili`, `Stripe`, `Trixie`},
		{`Bob`, `Chris`, `Mort`},
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
	s.Add(`F`, `E`) // F is a leaf via E not part of the cycle

	t.Log(s.ToMermaid()) // Should not panic

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

	cycles := s.GetCycles()
	sort.Strings(cycles)
	exp := []string{`A`, `B`, `C`}
	if diff := cmp.Diff(cycles, exp); len(diff) > 0 {
		t.Errorf("unexpected cycles (-got +exp):\n%s", diff)
	}
}
