package template

import (
	"cmp"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"time"

	"golang.org/x/text/feature/plural"
	"golang.org/x/text/language"
)

// See ".message-format-wg/spec/registry.xml".

type Func func(input *ResolvedValue, options Options, locale language.Tag) (output *ResolvedValue, err error)

type Registry map[string]Func

// Options are a possible options for the function.
type Options map[string]*ResolvedValue

// GetString returns the value by name.
// If the value is not found, returns the fallback value.
// If the value is not in allowed list, return error.
func (o Options) GetString(name, fallback string, validate ...Validate[string]) (string, error) {
	errorf := func(format string, args ...any) (string, error) {
		return "", fmt.Errorf(`get string option "%s": `+format, append([]any{name}, args...)...)
	}

	v, ok := o[name]
	if !ok {
		return fallback, nil
	}

	s, ok := v.value.(string)
	if !ok {
		return errorf("got %T", v)
	}

	for _, f := range validate {
		err := f(s)
		if err != nil {
			return errorf("%w", err)
		}
	}

	return s, nil
}

// GetInt returns the value by name.
// If the value is not found, returns the fallback value.
// If the value is not in allowed list, return error.
func (o Options) GetInt(name string, fallback int, validate ...Validate[int]) (int, error) {
	errorf := func(format string, args ...any) (int, error) {
		return 0, fmt.Errorf(`get int option "%s": `+format, append([]any{name}, args...)...)
	}

	v, ok := o[name]
	if !ok {
		return fallback, nil
	}

	var i int

	switch n := v.value.(type) {
	default:
		var err error

		i, err = castAs[int](n)
		if err != nil {
			return errorf("%w", err)
		}
	case string:
		n64, err := strconv.ParseInt(n, 10, 64)
		if err != nil {
			return 0, fmt.Errorf(`parse integer from string "%s": %w`, n, err)
		}

		i = int(n64)
	case int:
		i = n
	}

	for _, f := range validate {
		err := f(i)
		if err != nil {
			return errorf("%w", err)
		}
	}

	return i, nil
}

// NewRegistry returns a new registry with default functions.
func NewRegistry() Registry {
	return Registry{
		"date":     dateFunc,
		"datetime": datetimeFunc,
		"integer":  integerFunc,
		"number":   numberFunc,
		"string":   stringFunc,
		"time":     timeFunc,
	}
}

type Validate[T any] func(T) error

func oneOf[T comparable](values ...T) func(T) error {
	return func(value T) error {
		if len(values) == 0 {
			return nil
		}

		if slices.Contains(values, value) {
			return nil
		}

		return fmt.Errorf("want one of %v, got %v", values, value)
	}
}

func eqOrGreaterThan[T cmp.Ordered](v T) func(T) error {
	return func(value T) error {
		if value < v {
			return fmt.Errorf("want greater than %v, got %v", v, value)
		}

		return nil
	}
}

// castAs tries to cast any value to the given type.
func castAs[T any](val any) (T, error) {
	var zeroVal T

	typ := reflect.TypeOf(zeroVal)

	v := (reflect.ValueOf(val))
	if !v.Type().ConvertibleTo(typ) {
		return zeroVal, fmt.Errorf("convert %v to %T", v.Type(), zeroVal)
	}

	v = v.Convert(typ)

	return v.Interface().(T), nil //nolint:forcetypeassert
}

// getTZ gets the timezone information from the registry function options.
func getTZ(options Options) (*time.Location, error) {
	v, ok := options["timeZone"]
	if !ok {
		return time.UTC, nil
	}

	switch tz := v.value.(type) {
	default:
		return nil, fmt.Errorf("want timeZone as string or *time.Location, got %T", v.value)
	case *time.Location:
		return tz, nil
	case string:
		timezone, err := time.LoadLocation(tz)
		if err != nil {
			return nil, fmt.Errorf("load TZ data for %s: %w", tz, err)
		}

		return timezone, nil
	}
}

// pluralFormString formats plural.Form as string.
func pluralFormString(f plural.Form) string {
	switch f {
	default:
		return "other"
	case plural.Zero:
		return "zero"
	case plural.One:
		return "one"
	case plural.Two:
		return "two"
	case plural.Few:
		return "few"
	case plural.Many:
		return "many"
	}
}
