package two

import _ "unsafe" // for go:linkname

func init() {
	// Avoid dead-code elimination.
	// TODO(nevkontakte): This should not be necessary.
	var _ = doInternalOne
}

func DoTwo() string {
	return "doing two"
}

// The function below can't be imported from the package one the normal way because
// that would create an import cycle.
//go:linkname doInternalOne github.com/gopherjs/gopherjs/tests/testdata/linkname/one.doInternalOne
func doInternalOne() string

func DoImportedOne() string {
	return "doing imported one: " + doInternalOne()
}
