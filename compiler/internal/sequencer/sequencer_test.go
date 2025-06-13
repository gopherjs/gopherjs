package sequencer

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSequencingStrings(t *testing.T) {
	s := New[string]()

	// TODO: Improve test input

	s.Add(`a`, `ant`, `apple`)
	s.Add(`p`, `pepper`, `apple`)
	s.Add(`t`, `cat`, `ant`, `catnip`)
	s.Add(`cat`, `catnip`)

	count := s.DepthCount()
	got := make([][]string, count)
	for i := 0; i < s.DepthCount(); i++ {
		got[i] = s.Group(i)
	}

	exp := [][]string{
		{`catnip`, `ant`, `apple`},
		{`cat`, `pepper`, `ant`},
		{`t`, `p`, `a`},
	}
	if diff := cmp.Diff(got, exp); len(diff) > 0 {
		fmt.Println(s.ToMermaid())
		t.Errorf("unexpected sequencing (-got +exp):\n%s", diff)
	}
}
