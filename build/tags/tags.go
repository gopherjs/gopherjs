package tags

import (
	"bufio"
	"go/build/constraint"
	"strings"
)

// Match looks for build constraints in the given source text and
// returns whether the file should be used according to the given build flags.
//
// The given tags are the command line tags such as  "js", "linux",
// or "go1.22". These tags should include GOOS and GOARCH.
// The source text only needs to be the lines of text from the source
// file before the package clause.
//
// Returns true if the file matches the flags or no constraint was found,
// false if no match or a constraint was malformed.
// If a build constraint is malformed or doesn't match, then the
// line for that constraint is also returned.
//
// Note: This is used by /tests/gorepo/run.go to evaluate if found tests
// should be tested or not. Typically this will not be needed since tags
// are checked by the go command.
//
// See https://pkg.go.dev/cmd/go#hdr-Build_constraints
// See https://go.googlesource.com/proposal/+/master/design/draft-gobuild.md
func Match(src string, tags ...string) (bool, string) {
	tm := newTagMap(tags)

	// Custom rule, treat js as equivalent to nacl.
	if tm.has(`js`) {
		tm.add(`nacl`)
	}

	scanner := bufio.NewScanner(strings.NewReader(src))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// check that the package clause hasn't been reached.
		if strings.HasPrefix(line, `package`) {
			break
		}

		exp, err := constraint.Parse(line)
		if err != nil && err.Error() != `not a build constraint` {
			// constraint was likely malformed.
			return false, line
		}

		// constraint was found, exit if it doesn't match.
		if exp != nil && !exp.Eval(tm.has) {
			return false, line
		}
	}

	// no constraint found or all constraints matched.
	return true, ``
}

type tagMap map[string]struct{}

func newTagMap(tags []string) tagMap {
	tm := make(tagMap, len(tags))
	for _, tag := range tags {
		tm.add(tag)
	}
	return tm
}

func (tm tagMap) add(tag string) {
	tm[tag] = struct{}{}
}

func (tm tagMap) has(tag string) bool {
	_, has := tm[tag]
	return has
}
