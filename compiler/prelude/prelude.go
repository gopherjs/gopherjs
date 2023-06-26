package prelude

import (
	_ "embed"

	"github.com/evanw/esbuild/pkg/api"
	log "github.com/sirupsen/logrus"
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
		KeepNames:         true,
		Charset:           api.CharsetUTF8,
		LegalComments:     api.LegalCommentsEndOfFile,
	})
	for _, w := range result.Warnings {
		log.Warnf("%d:%d: %s\n%s\n", w.Location.Line, w.Location.Column, w.Text, w.Location.LineText)
	}
	if errCount := len(result.Errors); errCount > 0 {
		for _, e := range result.Errors {
			log.Errorf("%d:%d: %s\n%s\n", e.Location.Line, e.Location.Column, e.Text, e.Location.LineText)
		}
		log.Fatalf("Prelude minification failed with %d errors", errCount)
	}
	return string(result.Code)
}
