package template_test

import (
	"fmt"
	"os"

	"go.expect.digital/mf2"
	"go.expect.digital/mf2/template"
	"golang.org/x/text/language"
)

func ExampleTemplate_plainText() {
	// Define a MF2 string.
	const input = "Hello World!"

	// Parse template.
	t, err := template.New().Parse(input)
	if err != nil {
		panic(err)
	}

	err = t.Execute(os.Stdout, nil)
	if err != nil {
		panic(err)
	}

	// Output: Hello World!
}

func ExampleTemplate_simpleMessage() {
	// Define a MF2 string.
	const input = "Today is { $degrees :number signDisplay=always } degrees outside."

	// Parse template.
	t, err := template.New().Parse(input)
	if err != nil {
		panic(err)
	}

	// Execute the template.
	err = t.Execute(os.Stdout, map[string]any{"degrees": 15})
	if err != nil {
		panic(err)
	}

	// Output: Today is +15 degrees outside.
}

func ExampleTemplate_complexMessage() {
	// Define a MF2 string.
	const input = `.local $age = { 42 }
.input { $color :color style=RGB}
{{John is { $age } years old and his favorite color is { $color }.}}`

	color := func(value *template.ResolvedValue, options template.Options, _ language.Tag) (*template.ResolvedValue, error) { //nolint:lll
		errorf := func(format string, args ...any) (*template.ResolvedValue, error) {
			return nil, fmt.Errorf("exec color function: "+format, args...)
		}

		if value == nil {
			return errorf("input is required: %w", mf2.ErrBadOperand)
		}

		color := value.String()

		format := func() string {
			if len(options) == 0 {
				return color
			}

			style, err := options.GetString("style", "RGB")
			if err != nil {
				return color
			}

			var result string

			switch style {
			case "RGB":
				switch color {
				case "red":
					result = "255,0,0"
				case "green":
					result = "0,255,0"
				case "blue":
					result = "0,0,255"
				}
			case "HEX": // Other Implementations
			case "HSL": // Other Implementations
			}

			return result
		}

		return template.NewResolvedValue(color, template.WithFormat(format)), nil
	}

	// Parse template.
	t, err := template.New(template.WithFunc("color", color)).Parse(input)
	if err != nil {
		panic(err)
	}

	// Execute the template.
	err = t.Execute(os.Stdout, map[string]any{"color": "red"})
	if err != nil {
		panic(err)
	}

	// Output: John is 42 years old and his favorite color is 255,0,0.
}

// TODO(mvilks): come up with a good example of the ResolvedValue usage that requires access to the raw value.
// E.g. function ":parity" that returns a localized name for "odd"/"even".
