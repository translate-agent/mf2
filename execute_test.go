package mf2

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_Execute(t *testing.T) {
	t.Parallel()

	funcs := FuncMap{
		"jsonString": func(v any) (string, error) {
			b, err := json.Marshal(v)
			return string(b), err
		},
		"date": func(v string, inFormat, outFormat string) (string, error) {
			t, err := time.Parse(inFormat, v)

			return t.Format(outFormat), err
		},
		"number": func(v int, style string) (string, error) {
			if style == "percent" {
				return strconv.Itoa(v) + "%", nil
			}

			return strconv.Itoa(v), nil
		},
		"person": func(v *string, prefix string, titleCase bool) (string, error) {
			s := "John Doe"

			if v != nil {
				s = *v
			}

			if titleCase {
				s = strings.ToUpper(s)
			}

			if prefix != "" {
				s = prefix + " " + s
			}

			return s, nil
		},
	}

	// TODO: test markup, reserved-annotation, private-use-annotation, complex-messages

	for _, test := range []struct {
		name     string
		template *Template
		data     any
		expected string
	}{
		// {
		// 	name:     "simple message, text only",
		// 	template: NewTemplate("test1").MustParse("Hello, World!"),
		// 	data:     nil,
		// 	expected: "Hello, World!",
		// },
		// {
		// 	name:     "simple message, text with escaped chars",
		// 	template: NewTemplate("test2").MustParse("Hello, \\{World!\\}"),
		// 	data:     nil,
		// 	expected: "Hello, {World!}",
		// },
		// {
		// 	name:     "simple message, text with variable expr",
		// 	template: NewTemplate("test3").MustParse("Hello, { $var } World!"),
		// 	data:     map[string]any{"$var": "Wast"},
		// 	expected: "Hello, Wast World!",
		// },
		// {
		// 	name:     "simple message, text with multiple variable expr",
		// 	template: NewTemplate("test4").MustParse("You say, { $var1 } and I say, { $var2 }, { $var3 }!"),
		// 	data:     map[string]any{"$var1": "Goodbye", "$var2": "Hello", "$var3": "jello"},
		// 	expected: "You say, Goodbye and I say, Hello, jello!",
		// },
		// {
		// 	name:     "simple message, text with variable expr with function",
		// 	template: NewTemplate("test5").Funcs(funcs).MustParse("Person: { $var :jsonString }"),
		// 	data:     map[string]any{"$var": struct{ Name, LastName string }{"David", "Malkovich"}},
		// 	expected: `Person: {"Name":"David","LastName":"Malkovich"}`,
		// },
		// {
		// 	name: "simple message, text with variable expr (func with option)",
		// 	template: NewTemplate("test6").Funcs(funcs).
		// 		MustParse("Today is { $var :date inFormat = |2006-01-02| outFormat = |02 Jan 06| }!"),
		// 	data:     map[string]any{"$var": "2049-05-01"},
		// 	expected: "Today is 01 May 49!",
		// },
		{
			name: "simple message, text with variable expr (func with option and ref to var)",
			template: NewTemplate("test6").Funcs(funcs).
				MustParse("Today is { $var :date inFormat = |2006-01-02| outFormat = $outFormat }!"),
			data:     map[string]any{"$var": "2049-05-01", "$outFormat": "02 Jan 06"},
			expected: "Today is 01 May 49!",
		},
		// {
		// 	name:     "simple message, text with annotation expr",
		// 	template: NewTemplate("test7").Funcs(funcs).MustParse("Hello, { :person prefix = |Dr| title:Case = true }!"),
		// 	data:     nil,
		// 	expected: "Hello, Dr John Doe!",
		// },
		// {
		// 	name:     "simple message, text with literal expression (func with option)",
		// 	template: NewTemplate("test8").Funcs(funcs).MustParse("Color saturation: { 63 :number style = percent }"),
		// 	data:     nil,
		// 	expected: "Color saturation: 63%",
		// },
		// {
		// 	name: "simple message, text with quoted literal expression (func with option)",
		// 	template: NewTemplate("test9").Funcs(funcs).
		// 		MustParse("Hello, { |david malkovich| :person prefix = |Mr] titleCase = true }"),
		// 	data:     nil,
		// 	expected: "Hello, Mr David Malkovich!",
		// },
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
