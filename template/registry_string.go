package template

import (
	"fmt"

	"golang.org/x/text/language"
)

// stringFunc is the implementation of the string function.
// Formatting of strings as a literal and selection based on string equality.
func stringFunc(operand any, options Options, locale language.Tag) (*ResolvedValue, error) {
	errorf := func(format string, args ...any) (*ResolvedValue, error) {
		return nil, fmt.Errorf("exec string function: "+format, args...)
	}

	if operand == nil {
		return NewResolvedValue(""), nil
	}

	if len(options) > 0 {
		return errorf("want no options")
	}

	format := func() string {
		switch value := operand.(type) {
		default:
			s, err := castAs[string](operand) // if underlying type is not string, return empty string
			if err != nil {
				return ""
			}

			return s
		case fmt.Stringer:
			return value.String()
		case string, []byte, []rune, int, int8, int16, int32, int64,
			uint, uint8, uint16, uint32, uint64, float32, float64, bool,
			complex64, complex128, error:
			return fmt.Sprint(value)
		}
	}

	return NewResolvedValue(operand, WithFormat(format)), nil
}
