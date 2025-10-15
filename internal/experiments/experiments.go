// Package experiments managed the list of experimental feature flags supported
// by GopherJS.
//
// GOPHERJS_EXPERIMENT environment variable can be used to control which features
// are enabled.
package experiments

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
)

var (
	// ErrInvalidDest is a kind of error returned by parseFlags() when the dest
	// argument does not meet the requirements.
	ErrInvalidDest = errors.New("invalid flag struct")
	// ErrInvalidFormat is a kind of error returned by parseFlags() when the raw
	// flag string format is not valid.
	ErrInvalidFormat = errors.New("invalid flag string format")
)

// Env contains experiment flag values from the GOPHERJS_EXPERIMENT
// environment variable.
var Env Flags

func init() {
	if err := parseFlags(os.Getenv("GOPHERJS_EXPERIMENT"), &Env); err != nil {
		panic(fmt.Errorf("failed to parse GOPHERJS_EXPERIMENT flags: %w", err))
	}
}

// Flags contains flags for currently supported experiments.
type Flags struct {
	// e.g. Generics bool `flag:"generics"`
}

// parseFlags parses the `raw` flags string and populates flag values in the
// `dest`.
//
// `raw` is a comma-separated experiment flag list: `<flag1>,<flag2>,...`. Each
// flag may be either `<name>` or `<name>=<value>`. Omitting value is equivalent
// to "<name> = true". Spaces around name and value are trimmed during
// parsing. Flag name can't be empty. If the same flag is specified multiple
// times, the last instance takes effect.
//
// `dest` must be a pointer to a struct, which fields will be populated with
// flag values. Mapping between flag names and fields is established with the
// `flag` field tag. Fields without a flag tag will be left unpopulated.
// If multiple fields are associated with the same flag result is unspecified.
//
// Flags that don't have a corresponding field are silently ignored. This is
// done to avoid fatal errors when an experiment flag is removed from code, but
// remains specified in user's environment.
//
// Currently only boolean flag values are supported, as defined by
// `strconv.ParseBool()`.
func parseFlags(raw string, dest any) error {
	ptr := reflect.ValueOf(dest)
	if ptr.Type().Kind() != reflect.Pointer || ptr.Type().Elem().Kind() != reflect.Struct {
		return fmt.Errorf("%w: must be a pointer to a struct", ErrInvalidDest)
	}
	if ptr.IsNil() {
		return fmt.Errorf("%w: must not be nil", ErrInvalidDest)
	}
	fields := fieldMap(ptr.Elem())

	if raw == "" {
		return nil
	}
	entries := strings.Split(raw, ",")

	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		var key, val string
		if idx := strings.IndexRune(entry, '='); idx != -1 {
			key = strings.TrimSpace(entry[0:idx])
			val = strings.TrimSpace(entry[idx+1:])
		} else {
			key = entry
			val = "true"
		}

		if key == "" {
			return fmt.Errorf("%w: empty flag name", ErrInvalidFormat)
		}

		field, ok := fields[key]
		if !ok {
			// Unknown field value, possibly an obsolete experiment, ignore it.
			continue
		}
		if field.Type().Kind() != reflect.Bool {
			return fmt.Errorf("%w: only boolean flags are supported", ErrInvalidDest)
		}
		b, err := strconv.ParseBool(val)
		if err != nil {
			return fmt.Errorf("%w: can't parse %q as boolean for flag %q", ErrInvalidFormat, val, key)
		}
		field.SetBool(b)
	}

	return nil
}

// fieldMap returns a map of struct fieldMap keyed by the value of the "flag" tag.
//
// `s` must be a struct. Fields without a "flag" tag are ignored. If multiple
// fieldMap have the same flag, the last field wins.
func fieldMap(s reflect.Value) map[string]reflect.Value {
	typ := s.Type()
	result := map[string]reflect.Value{}
	for i := 0; i < typ.NumField(); i++ {
		if val, ok := typ.Field(i).Tag.Lookup("flag"); ok {
			result[val] = s.Field(i)
		}
	}
	return result
}
