package prelude

import (
	_ "embed"
)

//go:generate go run genmin.go

// Prelude is the GopherJS JavaScript interop layer.
var Prelude = prelude + numeric + types + goroutines + jsmapping

//go:embed prelude.js
var prelude string
