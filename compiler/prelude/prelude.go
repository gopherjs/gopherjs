package prelude

import (
	_ "embed"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/js"
)

// Prelude is the GopherJS JavaScript interop layer.
var Prelude = prelude + numeric + types + goroutines + jsmapping

//go:embed prelude.js
var prelude string

//go:embed types.js
var types string

//go:embed numeric.js
var numeric string

//go:embed jsmapping.js
var jsmapping string

//go:embed goroutines.js
var goroutines string

// Minified returns the minified JavaScript prelude. It panics if it encounters
// an error.
func Minified() string {
	m := minify.New()
	m.AddFunc("application/javascript", js.Minify)
	s, err := m.String("application/javascript", Prelude)
	if err != nil {
		panic(err)
	}
	return s
}
