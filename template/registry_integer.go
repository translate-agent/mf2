package template

import (
	"fmt"
	"maps"
	"math"

	"go.expect.digital/mf2"
	"golang.org/x/text/language"
)

// integerFunc is the implementation of the integer function. Locale-sensitive integer formatting.
func integerFunc(operand *ResolvedValue, options Options, locale language.Tag) (*ResolvedValue, error) {
	errorf := func(format string, args ...any) (*ResolvedValue, error) {
		return nil, fmt.Errorf("%w: exec integer function: "+format, append([]any{mf2.ErrBadOption}, args...)...)
	}

	validate := oneOf("minimumIntegerDigits", "maximumSignificantDigits", "signDisplay", "useGrouping", "select")

	for k := range options {
		err := validate(k)
		if err != nil {
			return errorf("%w", err)
		}
	}

	options = maps.Clone(options)
	if options == nil {
		options = make(Options)
	}

	options["maximumFractionDigits"] = NewResolvedValue(0)

	if operand != nil && operand.options != nil {
		cp := *operand
		cp.options = make(Options)

		for k, v := range operand.options {
			if validate(k) == nil {
				cp.options[k] = v
			}
		}

		operand = &cp
	}

	value, err := numberFunc(operand, options, locale)
	if err != nil {
		if value == nil {
			return nil, fmt.Errorf("exec integer func: %w", err)
		}

		value.function = ":integer"

		if v, ok := value.value.(float64); ok {
			value.value = math.Trunc(v)
		}

		return value, fmt.Errorf("exec integer func: %w", err)
	}

	value.function = ":integer"

	if v, ok := value.value.(float64); ok {
		value.value = math.Trunc(v)
	}

	return value, nil
}
