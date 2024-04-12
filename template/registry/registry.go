package registry

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"golang.org/x/text/language"
)

type F struct {
	Match  func(input any, options Opts, locale language.Tag) (output any, err error)
	Format func(input any, options Opts, locale language.Tag) (output any, err error)
}

type Registry map[string]F

// Func is a function that can be used in formatting and matching contexts.
type Func struct {
	FormatSignature, MatchSignature *Signature // Function signature when called in formatting or matching context
	Func                            func(any, map[string]any, language.Tag) (any, error)
}

// Signature is a signature of the function, i.e. what input and options are allowed.
type Signature struct {
	ValidateInput func(any) error
	Options       Options
}

// Option is a possible options for the function.
type Option struct {
	Name    string
	Default any

	ValidateValue  func(any) error // If option value is not restricted by a set of values.
	PossibleValues []any           // If option value is restricted by a set of values.
}

type Options []Option

type Opts map[string]any

// GetString returns the value by name.
// If the value is not found, returns the fallback value.
// If the value is not in allowed list, return error.
func (o Opts) GetString(name, fallback string, allowed []string) (string, error) {
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
func (o Opts) GetPositiveInt(name string, fallback int, allowed []int) (int, error) {
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
		"number":   {Format: numberRegistryFunc.Format, Match: numberRegistryFunc.Match},
		"datetime": {Format: datetimeRegistryFunc.Format, Match: datetimeRegistryFunc.Match},
	}
}

func (f *Func) Format(input any, options Opts, locale language.Tag) (any, error) {
	if err := f.FormatSignature.check(input, options); err != nil {
		return "", fmt.Errorf("check input: %w", err)
	}

	return f.Func(input, options, locale)
}

func (f *Func) Match(input any, options Opts, locale language.Tag) (any, error) {
	if err := f.MatchSignature.check(input, options); err != nil {
		return "", fmt.Errorf("check input: %w", err)
	}

	return f.Func(input, options, locale)
}

func (s *Signature) check(input any, options map[string]any) error {
	if len(s.Options) == 0 && len(options) > 0 {
		return errors.New("options are not allowed")
	}

	if s.ValidateInput != nil {
		if err := s.ValidateInput(input); err != nil {
			return fmt.Errorf("validate input: %w", err)
		}
	}

	if err := s.Options.check(options); err != nil {
		return fmt.Errorf("check options: %w", err)
	}

	return nil
}

func (o Options) check(got map[string]any) error {
	for name, val := range got {
		opt, ok := o.find(name)
		if !ok {
			validOptions := make([]string, 0, len(o))
			for _, opt := range o {
				validOptions = append(validOptions, opt.Name)
			}

			return fmt.Errorf("expected one of options - %s, got %s", strings.Join(validOptions, ", "), name)
		}

		if opt.ValidateValue != nil {
			if err := opt.ValidateValue(val); err != nil {
				return fmt.Errorf("validate option value '%s': %w", name, err)
			}
		}

		if len(opt.PossibleValues) > 0 {
			var found bool

			for _, possible := range opt.PossibleValues {
				if possible == val {
					found = true
					break
				}
			}

			if !found {
				return fmt.Errorf("invalid value '%s' for option '%s'. Valid values: %s  ", val, name, opt.PossibleValues)
			}
		}
	}

	// TODO: Append option to the got map, when option is not provided, but it has a default value.

	return nil
}

func (o Options) find(name string) (Option, bool) {
	for _, opt := range o {
		if opt.Name == name {
			return opt, true
		}
	}

	return Option{}, false
}
