package template_test

import (
	"fmt"
	"os"
	"slices"
	"strconv"

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
	// Define an MF2 complex message with input and local declarations.
	const input = `.input { $temp :number signDisplay=always }
.local $city = { Oslo }
{{The current temperature in {$city} is {$temp} degrees.}}`

	t, err := template.New().Parse(input)
	if err != nil {
		panic(err)
	}

	err = t.Execute(os.Stdout, map[string]any{"temp": 18})
	if err != nil {
		panic(err)
	}

	// Output: The current temperature in Oslo is +18 degrees.
}

func ExampleTemplate_match() {
	// Define an MF2 complex message with variant selection (.match).
	const input = `.input { $count :number }
.match $count
one {{{$count} item}}
*   {{{$count} items}}`

	t, err := template.New().Parse(input)
	if err != nil {
		panic(err)
	}

	err = t.Execute(os.Stdout, map[string]any{"count": 1})
	if err != nil {
		panic(err)
	}

	// Output: 1 item
}

func ExampleTemplate_Sprint() {
	const input = "Hello, {$name}!"

	t, err := template.New().Parse(input)
	if err != nil {
		panic(err)
	}

	msg, err := t.Sprint(map[string]any{"name": "Alice"})
	if err != nil {
		panic(err)
	}

	fmt.Println(msg)

	// Output: Hello, Alice!
}

func ExampleWithLocale() {
	const input = "Total: {$amount :number maximumFractionDigits=2}"

	t, err := template.New(template.WithLocale(language.German)).Parse(input)
	if err != nil {
		panic(err)
	}

	msg, err := t.Sprint(map[string]any{"amount": 1234.5})
	if err != nil {
		panic(err)
	}

	fmt.Println(msg)

	// Output: Total: 1.234,5
}

func ExampleWithFunc() {
	// Custom formatting function that formats color names into HEX or RGB.
	color := func(
		value *template.ResolvedValue,
		options template.Options,
		_ language.Tag,
	) (*template.ResolvedValue, error) {
		if value == nil {
			return nil, fmt.Errorf("input is required: %w", mf2.ErrBadOperand)
		}

		colorName := value.String()

		format := func() string {
			style, err := options.GetString("style", "RGB")
			if err != nil {
				return colorName
			}

			switch style {
			case "RGB":
				switch colorName {
				case "red":
					return "255, 0, 0"
				case "green":
					return "0, 255, 0"
				case "blue":
					return "0, 0, 255"
				}
			case "HEX":
				switch colorName {
				case "red":
					return "#FF0000"
				case "green":
					return "#00FF00"
				case "blue":
					return "#0000FF"
				}
			}

			return colorName
		}

		return template.NewResolvedValue(colorName, template.WithFormat(format)), nil
	}

	const input = "Favorite color: { $color :color style=HEX }."

	t, err := template.New(template.WithFunc("color", color)).Parse(input)
	if err != nil {
		panic(err)
	}

	msg, err := t.Sprint(map[string]any{"color": "red"})
	if err != nil {
		panic(err)
	}

	fmt.Println(msg)

	// Output: Favorite color: #FF0000.
}

func ExampleWithSelectKey() {
	// Custom selector function that matches even or odd numbers in .match expressions.
	parity := func(operand *template.ResolvedValue, _ template.Options, _ language.Tag) (*template.ResolvedValue, error) {
		if operand == nil {
			return nil, fmt.Errorf("operand is required: %w", mf2.ErrBadOperand)
		}

		num, err := strconv.Atoi(operand.String())
		if err != nil {
			return nil, fmt.Errorf("%w: %w", mf2.ErrBadOperand, err)
		}

		selectKey := func(keys []string) string {
			key := "odd"
			if num%2 == 0 {
				key = "even"
			}

			if slices.Contains(keys, key) {
				return key
			}

			return ""
		}

		return template.NewResolvedValue(operand, template.WithSelectKey(selectKey)), nil
	}

	const input = `.input { $count }
.local $p = { $count :parity }
.match $p
even {{Count {$count} is even}}
odd  {{Count {$count} is odd}}
*    {{Count {$count} is other}}`

	t, err := template.New(template.WithFunc("parity", parity)).Parse(input)
	if err != nil {
		panic(err)
	}

	msg, err := t.Sprint(map[string]any{"count": 42})
	if err != nil {
		panic(err)
	}

	fmt.Println(msg)

	// Output: Count 42 is even
}
