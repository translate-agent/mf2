package template

import (
	"fmt"
	"slices"

	"golang.org/x/text/language"
	"golang.org/x/text/unicode/norm"
)

// stringFunc is the implementation of the string function.
// Formatting of strings as a literal and selection based on string equality.
func stringFunc(operand *ResolvedValue, options Options, _ language.Tag) (*ResolvedValue, error) {
	errorf := func(format string, args ...any) (*ResolvedValue, error) {
		return nil, fmt.Errorf("exec string function: "+format, args...)
	}

	if len(options) > 0 {
		return errorf("want no options")
	}

	format := func() string {
		return defaultFormat(operand.value)
	}

	selectKey := func(keys []string) string {
		value := norm.NFC.String(format())

		if slices.Contains(keys, value) {
			return value
		}

		return ""
	}

	return NewResolvedValue(operand, WithFormat(format), WithSelectKey(selectKey)), nil
}
