package mf2

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_Execute(t *testing.T) {
	t.Parallel()

	funcs := FuncMap{
		"returnInt": func(_ any, _ map[string]any) (any, error) {
			return 43, nil
		},
		"returnString": func(_ any, _ map[string]any) (any, error) {
			return "Hello, World!", nil
		},
		"date": func(v any, params map[string]any) (any, error) {
			inFormat, _ := params["inFormat"].(string)
			outFormat, _ := params["outFormat"].(string)

			switch v := v.(type) {
			case string:
				t, err := time.Parse(inFormat, v)
				return t.Format(outFormat), err
			case time.Time:
				return v.Format(outFormat), nil
			default:
				return nil, fmt.Errorf("invalid type: %T", v)
			}
		},
		"json": func(v any, _ map[string]any) (any, error) {
			b, err := json.Marshal(v)
			return string(b), err
		},
		"number": func(v any, params map[string]any) (any, error) {
			switch v := v.(type) {
			case float64:
				if style, ok := params["style"]; ok && style == "percent" {
					return fmt.Sprintf("%.0f%%", v), nil
				}

				return fmt.Sprintf("%.0f", v), nil
			default:
				return nil, fmt.Errorf("invalid type: %T", v)
			}
		},
		"person": func(v any, params map[string]any) (any, error) { //nolint:unparam
			details := "john doe"

			if v, ok := v.(string); ok {
				details = v
			}

			if titleCase, ok := params["titleCase"].(string); ok && titleCase == "true" {
				details = strings.ToTitle(details)
			}

			if prefix, ok := params["prefix"].(string); ok {
				details = prefix + " " + details
			}

			return details, nil
		},
	}

	// TODO: test markup, reserved-annotation, private-use-annotation, complex-messages

	for _, test := range []struct {
		name     string
		template *Template
		data     any
		expected string
	}{
		{
			name:     "simple message, text only",
			template: NewTemplate("test1").MustParse("Hello, World!"),
			data:     nil,
			expected: "Hello, World!",
		},
		{
			name:     "simple message, text with escaped chars",
			template: NewTemplate("test2").MustParse("Hello, \\{World!\\}"),
			data:     nil,
			expected: "Hello, {World!}",
		},
		{
			name:     "simple message, text with variable expr",
			template: NewTemplate("test3").MustParse("Hello, { $var } World!"),
			data:     map[string]any{"$var": "Wast"},
			expected: "Hello, Wast World!",
		},
		{
			name:     "simple message, text with multiple variable expr",
			template: NewTemplate("test4").MustParse("You say, { $var1 } and I say, { $var2 }, { $var3 }!"),
			data:     map[string]any{"$var1": "Goodbye", "$var2": "Hello", "$var3": "jello"},
			expected: "You say, Goodbye and I say, Hello, jello!",
		},
		{
			name:     "simple message, text with variable expr with function",
			template: NewTemplate("test5").Funcs(funcs).MustParse("Person: { $name :json }"),
			data:     map[string]any{"$name": struct{ Name, LastName string }{"David", "Doe"}},
			expected: `Person: {"Name":"David","LastName":"Doe"}`,
		},
		{
			name: "simple message, text with variable expr (func with option)",
			template: NewTemplate("test6").Funcs(funcs).
				MustParse("Today is { $date :date inFormat = |2006-01-02| outFormat = |02 Jan 06| }!"),
			data:     map[string]any{"$date": "2049-05-01"},
			expected: "Today is 01 May 49!",
		},
		{
			name: "simple message, text with variable expr (func with option and ref to var)",
			template: NewTemplate("test6").Funcs(funcs).
				MustParse("Today is { $date :date inFormat = |2006-01-02| outFormat = $outFormat }!"),
			data:     map[string]any{"$date": "2049-05-01", "$outFormat": "02 Jan 06"},
			expected: "Today is 01 May 49!",
		},
		{
			name:     "simple message, text with annotation expr",
			template: NewTemplate("test7").Funcs(funcs).MustParse("Hello, { :person prefix = |Dr| title:Case = true }!"),
			data:     nil,
			expected: "Hello, Dr John Doe!",
		},
		{
			name:     "simple message, text with literal expression (func with option)",
			template: NewTemplate("test8").Funcs(funcs).MustParse("Color saturation: { 63 :number style = percent }"),
			data:     nil,
			expected: "Color saturation: 63%",
		},
		{
			name: "simple message, text with quoted literal expression (func with option)",
			template: NewTemplate("test9").Funcs(funcs).
				MustParse("Hello, { |david Doe| :person prefix = |Mr] titleCase = true }"),
			data:     nil,
			expected: "Hello, Mr David Doe!",
		},
	} {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var b strings.Builder

			require.NoError(t, test.template.Execute(&b, test.data))
			require.Equal(t, test.expected, b.String())
		})
	}
}
