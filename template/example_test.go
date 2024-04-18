package template_test

import (
	"fmt"
	"os"

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

	if err := t.Execute(os.Stdout, nil); err != nil {
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
	if err = t.Execute(os.Stdout, map[string]any{"degrees": 15}); err != nil {
		panic(err)
	}

	// Output: Today is +15 degrees outside.
}

func ExampleTemplate_complexMessage() {
	// Define a MF2 string.
	const input = `.local $age = { 42 }
.input { $color :color style=RGB}
{{John is { $age } years old and his favorite color is { $color }.}}`

	color := func(value any, options template.Options, locale language.Tag) (any, error) {
		if value == nil {
			return "", fmt.Errorf("input is required: %w", template.ErrOperandMismatch)
		}

		color, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("input is not a string: %w", template.ErrOperandMismatch)
		}

		if options == nil {
			return color, nil
		}

		style, err := options.GetString("style", "RGB")
		if err != nil {
			return nil, fmt.Errorf("get style: %w", err)
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

		return result, nil
	}
	// }

	// Parse template.
	t, err := template.New(template.WithFunc("color", template.RegistryFunc{Format: color})).Parse(input)
	if err != nil {
		panic(err)
	}

	// Execute the template.
	if err = t.Execute(os.Stdout, map[string]any{"color": "red"}); err != nil {
		panic(err)
	}

	// Output: John is 42 years old and his favorite color is 255,0,0.
}
