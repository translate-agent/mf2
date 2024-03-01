package registry

import (
	"errors"
	"fmt"
	"strings"
)

type Registry []Func

// Func is a function that can be used in formatting and matching contexts.
type Func struct {
	FormatSignature *Signature                             // Signature of the function when called in formatting context
	MatchSignature  *Signature                             // Signature of the function when called in matching context
	F               func(any, map[string]any) (any, error) // Function itself
	Name            string
	Description     string
}

// Signature is a signature of the function, i.e. what input and options are allowed.
type Signature struct {
	ValidateInput func(any) error // Only when input is true
	Options       Options         // Possible options for the function
	Input         bool            // If true, input is required
}

// Option is a possible options for the function.
type Option struct {
	Name        string
	Description string
	Default     any

	ValidateValue  func(any) error // If option value is not restricted by a set of values.
	PossibleValues []any           // If option value is restricted by a set of values.
}

type Options []Option

func NewRegistry() Registry {
	return []Func{
		*stringRegistryF,
		*numberRegistryF,
	}
}

func (r Registry) Find(name string) (Func, bool) {
	for _, fn := range r {
		if fn.Name == name {
			return fn, true
		}
	}

	return Func{}, false
}

func (r *Func) Format(input any, options map[string]any) (any, error) {
	if r.FormatSignature == nil {
		return "", fmt.Errorf("function '%s' is not allowed to use in formatting context", r.Name)
	}

	if err := r.FormatSignature.check(input, options); err != nil {
		return "", fmt.Errorf("check input: %w", err)
	}

	return r.F(input, options)
}

func (r *Func) Match(input any, options map[string]any) (any, error) {
	if r.MatchSignature == nil {
		return "", fmt.Errorf("function '%s' is not allowed to use in selector context", r.Name)
	}

	if err := r.MatchSignature.check(input, options); err != nil {
		return "", fmt.Errorf("check input: %w", err)
	}

	return r.F(input, options)
}

func (s *Signature) check(input any, options map[string]any) error {
	if s.Input && input == nil {
		return errors.New("input is nil, but required")
	}

	if !s.Input && input != nil {
		return errors.New("input is not allowed")
	}

	if s.Options == nil && len(options) > 0 {
		return errors.New("options are not allowed")
	}

	if s.ValidateInput != nil {
		if err := s.ValidateInput(input); err != nil {
			return fmt.Errorf("validate input: %w", err)
		}
	}

	if len(options) > 0 {
		if err := s.Options.check(options); err != nil {
			return fmt.Errorf("check options: %w", err)
		}
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

			return fmt.Errorf("unknown option: '%s'. Valid options: [%s]", name, strings.Join(validOptions, ", "))
		}

		if opt.ValidateValue != nil {
			if err := opt.ValidateValue(val); err != nil {
				return fmt.Errorf("validate options value '%s': %w", name, err)
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
