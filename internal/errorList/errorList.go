package errorList

import (
	"errors"
	"fmt"
)

// ErrTooManyErrors is added to the ErrorList by the Trim method.
var ErrTooManyErrors = errors.New("too many errors")

// ErrorList wraps multiple errors as a single error.
type ErrorList []error

func (errs ErrorList) Error() string {
	if len(errs) == 0 {
		return "<no errors>"
	}
	return fmt.Sprintf("%s (and %d more errors)", errs[0].Error(), len(errs[1:]))
}

// ErrOrNil returns nil if ErrorList is empty, or the error otherwise.
func (errs ErrorList) ErrOrNil() error {
	if len(errs) == 0 {
		return nil
	}
	return errs
}

// Append an error to the list.
//
// If err is an instance of ErrorList, the lists are concatenated together,
// otherwise err is appended at the end of the list. If err is nil, the list is
// returned unmodified.
//
//	err := DoStuff()
//	errList := errList.Append(err)
func (errs ErrorList) Append(err error) ErrorList {
	if err == nil {
		return errs
	}
	if err, ok := err.(ErrorList); ok {
		return append(errs, err...)
	}
	return append(errs, err)
}

// AppendDistinct is similar to Append, but doesn't append the error if it has
// the same message as the last error on the list.
func (errs ErrorList) AppendDistinct(err error) ErrorList {
	if l := len(errs); l > 0 {
		if prev := errs[l-1]; prev != nil && err.Error() == prev.Error() {
			return errs // The new error is the same as the last one, skip it.
		}
	}

	return errs.Append(err)
}

// Trim the error list if it has more than limit errors. If the list is trimmed,
// all extraneous errors are replaced with a single ErrTooManyErrors, making the
// returned ErrorList length of limit+1.
func (errs ErrorList) Trim(limit int) ErrorList {
	if len(errs) <= limit {
		return errs
	}

	return append(errs[:limit], ErrTooManyErrors)
}
