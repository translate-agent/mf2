package template

import (
	"fmt"

	"golang.org/x/text/language"
)

// See ".messager-format-wg/spec/registry.xml".

// stringRegistryFunc is the implementation of the string function.
// Formatting of strings as a literal and selection based on string equality.
var stringRegistryFunc = RegistryFunc{
	Format: stringFunc,
	Match:  stringFunc,
}

func stringFunc(operand any, options Options, locale language.Tag) (any, error) {
	errorf := func(format string, args ...any) (any, error) {
		return nil, fmt.Errorf("exec string function: "+format, args...)
	}

	if operand == nil {
		return "", nil
	}

	if len(options) > 0 {
		return errorf("want no options")
	}

	switch value := operand.(type) {
	default:
		s, err := castAs[string](operand) // if underlying type is not string, return error
		if err != nil {
			return errorf("unsupported operand type: %T: %w", operand, err)
		}

		return s, nil
	case fmt.Stringer:
		return value.String(), nil
	case string, []byte, []rune, int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64, float32, float64, bool,
		complex64, complex128, error:
		return fmt.Sprint(value), nil
	}
}
