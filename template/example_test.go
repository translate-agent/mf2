package template_test

import (
	"fmt"
	"os"
	"strings"

	"go.expect.digital/mf2/template"
)

func ExampleTemplate_plainText() {
	// Define a MF2 string.
	const input = "Hello World!"

	// Parse template.
	t, err := template.New().Parse(input)
	if err != nil {
		// Handle error.
	}

	if err := t.Execute(os.Stdout, nil); err != nil {
		// Handle error.
	}

	// Output: Hello World!
}

func ExampleTemplate_simpleMessage() {
	// Define a MF2 string.
	const input = "Hello, { $firstName :upper } { $lastName :lower style=first }!"

	// Parse template.
	t, err := template.New().Parse(input)
	if err != nil {
		// Handle error.
	}

	// Add functions to the template.
	t.AddFunc("upper", func(operand any, _ map[string]any) (string, error) {
		return strings.ToUpper(fmt.Sprint(operand)), nil
	})

	t.AddFunc("lower", func(operand any, options map[string]any) (string, error) {
		if options == nil {
			return strings.ToLower(fmt.Sprint(operand)), nil
		}

		if style, ok := options["style"].(string); ok && style == "first" {
			return strings.ToLower(fmt.Sprint(operand)[0:1]) + fmt.Sprint(operand)[1:], nil
		}

		return "", fmt.Errorf("bad options")
	})

	// Execute the template.
	if err = t.Execute(os.Stdout, map[string]any{"firstName": "John", "lastName": "DOE"}); err != nil {
		// Handle error.
	}

	// Output: Hello, JOHN dOE!
}

func ExampleTemplate_complexMessage() {
	// Define a MF2 string.
	const input = `.local $age = { 42 }
.input { $color :color style=RGB}
{{John is { $age } years old and his favorite color is { $color }.}}`

	// Parse template.
	t, err := template.New().Parse(input)
	if err != nil {
		// Handle error.
	}

	// Add functions to the template.
	t.AddFunc("color", func(color any, opts map[string]any) (string, error) {
		if opts == nil {
			return fmt.Sprint(color), nil
		}

		colorStr := fmt.Sprint(color)
		if style, ok := opts["style"].(string); ok && style == "RGB" {
			switch colorStr {
			case "red":
				return "255,0,0", nil
			case "green":
				return "0,255,0", nil
			case "blue":
				return "0,0,255", nil
			}
		}

		return "", fmt.Errorf("bad options")
	})

	// Execute the template.
	if err = t.Execute(os.Stdout, map[string]any{"color": "red"}); err != nil {
		// Handle error.
	}

	// Output: John is 42 years old and his favorite color is 255,0,0.
}
