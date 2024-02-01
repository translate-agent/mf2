package mf2

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_Execute(t *testing.T) {
	t.Parallel()

	//nolint:lll
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
			name:     "simple message, text escaped chars",
			template: NewTemplate("test2").MustParse("Hello, \\{World!\\}"),
			data:     nil,
			expected: "Hello, {World!}",
		},
		{
			name:     "simple message, text with variable",
			template: NewTemplate("test3").MustParse("Hello, { $var } World!"),
			data:     map[string]string{"$var": "Wast"},
			expected: "Hello, Wast World!",
		},
		{
			name:     "simple message, text with two variables",
			template: NewTemplate("test4").MustParse("You say, { $var1 } and I say, { $var2 }!"),
			data:     map[string]string{"$var1": "Goodbye", "$var2": "Hello"},
			expected: "You say, Goodbye and I say, Hello!",
		},
		{
			name:     "simple message, text and annotation expression", // CLARIFY: how to resolve annotation expressions?
			template: NewTemplate("test5").MustParse("Func result: { :foo }!"),
			data:     nil,
			expected: "Func result: bar!",
		},
		{
			name:     "simple message, variable with annotation",
			template: NewTemplate("test6").MustParse("Today is { $date :date format = \"02 Jan 06\" }!"),
			data:     map[string]string{"$date": "2049-05-01"},
			expected: "Today is 01 May 49!",
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
