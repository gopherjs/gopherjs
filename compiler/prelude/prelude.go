package prelude

import (
	_ "embed"
)

//go:generate go run genmin.go

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
