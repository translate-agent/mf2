package registry

import (
	"errors"
	"fmt"
	"slices"

	"golang.org/x/exp/constraints"
	"golang.org/x/text/language"
)

type Func struct {
	Match  func(input any, options Options, locale language.Tag) (output any, err error)
	Format func(input any, options Options, locale language.Tag) (output any, err error)
}

type Registry map[string]Func

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

	i, err := castAs[int](v)
	if err != nil {
		return 0, fmt.Errorf("convert val to int: %w", err)
	}

	if i < 0 {
		return 0, errors.New("value must be at least 0")
	}

	for _, f := range validate {
		if err := f(i); err != nil {
			return 0, err
		}
	}

	return i, nil
}

// New returns a new registry with default functions.
func New() Registry {
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
