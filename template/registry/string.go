package registry

import (
	"fmt"

	"golang.org/x/text/language"
)

// https://github.com/unicode-org/message-format-wg/blob/20a61b4af534acb7ecb68a3812ca0143b34dfc76/spec/registry.xml#L259

// stringRegistryFunc is the implementation of the string function.
// Formatting of strings as a literal and selection based on string equality.
var stringRegistryFunc = &Func{
	Name:            "string",
	FormatSignature: &Signature{IsInputRequired: true},
	MatchSignature:  &Signature{IsInputRequired: true},
	Func:            stringFunc,
}

func stringFunc(input any, _ map[string]any, locale language.Tag) (any, error) {
	switch v := input.(type) {
	case fmt.Stringer:
		return v.String(), nil
	case string, []byte, []rune, int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64, float32, float64, bool,
		complex64, complex128, error, nil:
		return fmt.Sprint(v), nil
	default:
		val, err := castAs[string](input) // if underlying type is not string, return error
		if err != nil {
			return nil, fmt.Errorf("unsupported type: %T: %w", input, err)
		}

		return val, nil
	}
}
