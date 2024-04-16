package registry

import (
	"errors"
	"fmt"
	"slices"

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
func (o Options) GetString(name, fallback string, allowed []string) (string, error) {
	v, ok := o[name]
	if !ok {
		return fallback, nil
	}

	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("expected string, got %T", v)
	}

	if len(allowed) > 0 && slices.Contains(allowed, s) {
		return s, nil
	}

	return "", fmt.Errorf("expected one of %s, got %s", allowed, s)
}

// GetPositiveInt returns the value by name.
// If the value is not found, returns the fallback value.
// If the value is not in allowed list, return error.
func (o Options) GetPositiveInt(name string, fallback int, allowed []int) (int, error) {
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

	if len(allowed) > 0 && !slices.Contains(allowed, i) {
		return 0, fmt.Errorf("expected one of %v, got %d", allowed, i)
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
