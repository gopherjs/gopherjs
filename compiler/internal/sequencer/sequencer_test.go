package sequencer

import (
	"fmt"
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

	count := s.DepthCount()
	got := make([][]string, count)
	for i := 0; i < s.DepthCount(); i++ {
		group := s.Group(i)
		sort.Strings(group)
		got[i] = group
	}

	exp := [][]string{
		{`Bingo`, `Bluey`, `Brandy`, `Muffin`, `Rad`, `Socks`},
		{`Bandit`, `Chili`, `Stripe`, `Trixie`},
		{`Bob`, `Chris`, `Mort`},
	}
	if diff := cmp.Diff(got, exp); len(diff) > 0 {
		fmt.Println(s.ToMermaid())
		t.Errorf("unexpected sequencing (-got +exp):\n%s", diff)
	}
}
