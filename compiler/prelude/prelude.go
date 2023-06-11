package prelude

import (
	_ "embed"
	"fmt"

	"github.com/evanw/esbuild/pkg/api"
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

func Minified() string {
	result := api.Transform(Prelude, api.TransformOptions{
		Target:            api.ES2015,
		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
		Charset:           api.CharsetUTF8,
		LegalComments:     api.LegalCommentsEndOfFile,
	})
	if len(result.Errors) > 0 {
		e := result.Errors[0]
		panic(fmt.Sprintf("%d:%d: %s\n%s\n", e.Location.Line, e.Location.Column, e.Text, e.Location.LineText))
	}
	return string(result.Code)

}
