package template

import (
	"fmt"

	"go.expect.digital/mf2"
	"golang.org/x/text/language"
	"golang.org/x/text/unicode/norm"
)

// stringFunc is the implementation of the string function.
// Formatting of strings as a literal and selection based on string equality.
func stringFunc(operand *ResolvedValue, options Options, _ language.Tag) (*ResolvedValue, error) {
	errorf := func(format string, args ...any) (*ResolvedValue, error) {
		return nil, fmt.Errorf("%w: exec string function: "+format, append([]any{mf2.ErrBadOption}, args...)...)
	}

	if len(options) > 0 {
		return errorf("want no options")
	}

	format := func() string {
		return defaultFormat(operand.value)
	}

	selectKey := func(keys []string) string {
		value := norm.NFC.String(format())

		for _, k := range keys {
			if norm.NFC.String(k) == value {
				return k
			}
		}

		return ""
	}

	return NewResolvedValue(operand, WithFormat(format), WithSelectKey(selectKey)), nil
}
