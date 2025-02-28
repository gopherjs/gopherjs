package sources

import (
	"go/types"

	"github.com/gopherjs/gopherjs/internal/errorList"
)

// errorCollectingImporter implements go/types.Importer interface and
// wraps it to collect import errors.
type errorCollectingImporter struct {
	Importer SourcesImporter
	Errors   errorList.ErrorList
}

func (ei *errorCollectingImporter) Import(path string) (*types.Package, error) {
	if path == "unsafe" {
		return types.Unsafe, nil
	}

	srcs, err := ei.Importer(path)
	if err != nil {
		ei.Errors = ei.Errors.AppendDistinct(err)
		return nil, err
	}
	return srcs.Package, nil
}
