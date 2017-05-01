package jslib

import (
	"fmt"
)

type ErrorMissingTarget struct{}

func (e ErrorMissingTarget) Error() string {
	return "target must not be <nil>"
}

type ErrorMissingReader struct{}

func (e ErrorMissingReader) Error() string {
	return "reader must not be <nil>"
}

type ErrorParsing struct {
	FileName, Message string
}

func (e ErrorParsing) Error() string {
	return fmt.Sprintf("can't parse file %#v: %s", e.FileName, e.Message)
}

type ErrorCompiling string

func (e ErrorCompiling) Error() string {
	return string(e)
}

type ErrorImportingDependencies string

func (e ErrorImportingDependencies) Error() string {
	return string(e)
}
