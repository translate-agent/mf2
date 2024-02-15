package template_test

import (
	"fmt"
	"os"
	"strings"

	"go.expect.digital/mf2/template"
)

func ExampleTemplate_simple() {
	// Define a MF2 string.
	const input = "Hello, { $name }!"

	// Parse template.
	t, err := template.New().Parse(input)
	if err != nil {
		// Handle error.
	}

	// Execute the template.
	if err = t.Execute(os.Stdout, map[string]any{"name": "World"}); err != nil {
		// Handle error.
	}

	// Output: Hello, World!
}

func ExampleTemplate_func() {
	// Define a MF2 string.
	const input = "Hello, { $name :upper }!"

	// Parse template.
	t, err := template.New().Parse(input)
	if err != nil {
		// Handle error.
	}

	// Add a function to the template.
	t.AddFunc("upper", func(operand any, _ map[string]any) (string, error) {
		return strings.ToUpper(fmt.Sprint(operand)), nil
	})

	// Execute the template.
	if err = t.Execute(os.Stdout, map[string]any{"name": "World"}); err != nil {
		// Handle error.
	}

	// Output: Hello, WORLD!
}
