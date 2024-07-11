package template

import (
	"fmt"
	"slices"
	"strconv"

	"golang.org/x/exp/constraints"
	"golang.org/x/text/language"
)

type RegistryFunc struct {
	Match  func(input any, options Options, locale language.Tag) (output any, err error)
	Format func(input any, options Options, locale language.Tag) (output any, err error)
}

type Registry map[string]RegistryFunc

// Options are a possible options for the function.
type Options map[string]any

// GetString returns the value by name.
// If the value is not found, returns the fallback value.
// If the value is not in allowed list, return error.
func (o Options) GetString(name, fallback string, validate ...Validate[string]) (string, error) {
	v, ok := o[name]
	if !ok {
		return fallback, nil
	}

	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("expected string, got %T", v)
	}

	for _, f := range validate {
		if err := f(s); err != nil {
			return "", err
		}
	}

	return s, nil
}

// GetInt returns the value by name.
// If the value is not found, returns the fallback value.
// If the value is not in allowed list, return error.
func (o Options) GetInt(name string, fallback int, validate ...Validate[int]) (int, error) {
	v, ok := o[name]
	if !ok {
		return fallback, nil
	}

	if s, ok := v.(string); ok {
		var err error

		v, err = strconv.ParseInt(s, 10, 32)
		if err != nil {
			return 0, fmt.Errorf("parse integer from string '%s': %w", s, err)
		}
	}

	i, err := castAs[int](v)
	if err != nil {
		return 0, err
	}

	for _, f := range validate {
		if err := f(i); err != nil {
			return 0, err
		}
	}

	return i, nil
}

// NewRegistry returns a new registry with default functions.
func NewRegistry() Registry {
	return Registry{
		"string":   stringRegistryFunc,
		"number":   numberRegistryFunc,
		"datetime": datetimeRegistryFunc,
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

		return fmt.Errorf("expected one of %v, got %v", values, value)
	}
}

func eqOrGreaterThan[T constraints.Ordered](min T) func(T) error {
	return func(value T) error {
		if value < min {
			return fmt.Errorf("expected value greater than %v, got %v", min, value)
		}

		return nil
	}
}
