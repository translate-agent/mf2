package template_test

import (
	"errors"
	"os"

	"go.expect.digital/mf2/template"
	"go.expect.digital/mf2/template/registry"
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

	// Parse template.
	t, err := template.New().Parse(input)
	if err != nil {
		panic(err)
	}

	// Define new function color
	colorF := registry.Func{
		Name: "color",
		FormatSignature: &registry.Signature{
			// Mark that input/operand is required for a function
			Input: true,
			// Set a validation function for the input/operand, in this
			// scenario we want to ensure that the input is a string
			ValidateInput: func(a any) error {
				if _, ok := a.(string); !ok {
					return errors.New("input is not a string")
				}
				return nil
			},
			// Define options for the function
			Options: registry.Options{
				{
					Name:           "style",
					Description:    `The style of the color.`,
					PossibleValues: []any{"RGB", "HEX", "HSL"}, // Define possible values for the option
					Default:        "RGB",                      // Set a default value for the option
				},
			},
		},
		// Define the function
		F: func(color any, options map[string]any) (any, error) {
			if options == nil {
				return color, nil
			}

			colorStr := color.(string) //nolint:forcetypeassert // Already validated by ValidateInput

			style, ok := options["style"].(string)
			if !ok {
				style = "RGB"
			}

			var result string

			switch style {
			case "RGB":
				switch colorStr {
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
		},
	}

	t.AddFunc(colorF)

	// Execute the template.
	if err = t.Execute(os.Stdout, map[string]any{"color": "red"}); err != nil {
		panic(err)
	}

	// Output: John is 42 years old and his favorite color is 255,0,0.
}
