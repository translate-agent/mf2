package template

import (
	"fmt"

	"golang.org/x/text/language"
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
		switch v := operand.value.(type) {
		default:
			// TODO(jhorsts): if underlying type is not string, return errorf("unsupported value type: %T: %w", r.value, err)
			s, _ := v.(string)
			return s
		case fmt.Stringer:
			return v.String()
		case nil:
			return ""
		case string, []byte, []rune, int, int8, int16, int32, int64,
			uint, uint8, uint16, uint32, uint64, float32, float64, bool,
			complex64, complex128, error:
			return fmt.Sprint(v)
		}
	}

	selectKey := func(keys []string) string {
		value := format()

		for _, key := range keys {
			if key == value {
				return key
			}
		}

		return ""
	}

	return NewResolvedValue(operand, WithFormat(format), WithSelectKey(selectKey)), nil
}
