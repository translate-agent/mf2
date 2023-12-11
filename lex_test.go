package mf2

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_lex(t *testing.T) {
	t.Parallel()

	var (
		tokenEOF             = mk(itemEOF, "")
		tokenExpressionOpen  = mk(itemExpressionOpen, "{")
		tokenExpressionClose = mk(itemExpressionClose, "}")
	)

	for _, test := range []struct {
		name     string
		input    string // MessageFormat2 formatted string
		expected []item
	}{
		{
			name:     "empty simple message",
			input:    "",
			expected: []item{tokenEOF},
		},
		{
			name:  "text",
			input: `escaped text: \\ \} \{`,
			expected: []item{
				mk(itemText, `escaped text: \ } {`),
				tokenEOF,
			},
		},
		{
			name:  "function",
			input: "{:rand}",
			expected: []item{
				tokenExpressionOpen,
				mk(itemFunction, "rand"),
				tokenExpressionClose,
				tokenEOF,
			},
		},

		{
			name:  "opening function",
			input: "{+button}",
			expected: []item{
				tokenExpressionOpen,
				mk(itemOpeningFunction, "button"),
				tokenExpressionClose,
				tokenEOF,
			},
		},
		{
			name:  "closing function",
			input: "{-button}",
			expected: []item{
				tokenExpressionOpen,
				mk(itemClosingFunction, "button"),
				tokenExpressionClose,
				tokenEOF,
			},
		},
		{
			name:  "opening and closing functions",
			input: "{+button}Submit{-button}",
			expected: []item{
				tokenExpressionOpen,
				mk(itemOpeningFunction, "button"),
				tokenExpressionClose,
				mk(itemText, "Submit"),
				tokenExpressionOpen,
				mk(itemClosingFunction, "button"),
				tokenExpressionClose,
				tokenEOF,
			},
		},
		{
			name:  "variable",
			input: "{$count :math:round}",
			expected: []item{
				tokenExpressionOpen,
				mk(itemVariable, "count"),
				mk(itemWhitespace, " "),
				mk(itemFunction, "math:round"),
				tokenExpressionClose,
				tokenEOF,
			},
		},
		{
			name:  "text with variable",
			input: "Hello, {$guest}!",
			expected: []item{
				mk(itemText, "Hello, "),
				tokenExpressionOpen,
				mk(itemVariable, "guest"),
				tokenExpressionClose,
				mk(itemText, "!"),
				tokenEOF,
			},
		},
		{
			name:  "empty quoted literal",
			input: "{||}",
			expected: []item{
				mk(itemExpressionOpen, "{"),
				mk(itemLiteral, ""),
				mk(itemExpressionClose, "}"),
				tokenEOF,
			},
		},
		{
			name:  "quoted literal",
			input: "{|Hello, world!| :uppercase}",
			expected: []item{
				tokenExpressionOpen,
				mk(itemLiteral, "Hello, world!"),
				mk(itemWhitespace, " "),
				mk(itemFunction, "uppercase"),
				tokenExpressionClose,
				tokenEOF,
			},
		},
		{
			name:  "number literal",
			input: "{-1.9e+10 :odd}",
			expected: []item{
				tokenExpressionOpen,
				mk(itemLiteral, "-1.9e+10"),
				mk(itemWhitespace, " "),
				mk(itemFunction, "odd"),
				tokenExpressionClose,
				tokenEOF,
			},
		},
		{
			name:  "unquoted literal",
			input: "{hello :uppercase}",
			expected: []item{
				tokenExpressionOpen,
				mk(itemLiteral, "hello"),
				mk(itemWhitespace, " "),
				mk(itemFunction, "uppercase"),
				tokenExpressionClose,
				tokenEOF,
			},
		},
		{
			name:  "reserved",
			input: `{!a @b #c %d *e <|hello| > /\{ ?\| ~\}}`,
			expected: []item{
				tokenExpressionOpen,
				mk(itemReserved, "!a"),
				mk(itemWhitespace, " "),
				mk(itemReserved, "@b"),
				mk(itemWhitespace, " "),
				mk(itemReserved, "#c"),
				mk(itemWhitespace, " "),
				mk(itemReserved, "%d"),
				mk(itemWhitespace, " "),
				mk(itemReserved, "*e"),
				mk(itemWhitespace, " "),
				mk(itemReserved, "<"),
				mk(itemLiteral, "hello"),
				mk(itemWhitespace, " "),
				mk(itemReserved, ">"),
				mk(itemWhitespace, " "),
				mk(itemReserved, `/{`),
				mk(itemWhitespace, " "),
				mk(itemReserved, "?|"),
				mk(itemWhitespace, " "),
				mk(itemReserved, "~}"),
				tokenExpressionClose,
				tokenEOF,
			},
		},
		{
			name:  "private use", // TODO: incomplete
			input: "{^ &}",
			expected: []item{
				tokenExpressionOpen,
				mk(itemPrivate, "^"),
				mk(itemWhitespace, " "),
				mk(itemPrivate, "&"),
				tokenExpressionClose,
				tokenEOF,
			},
		},
		{
			name:  "local declaration",
			input: ".local $hostName = {$host} .local $h = {$host}",
			expected: []item{
				// .local $hostName = {$host}
				mk(itemKeyword, "local"),
				mk(itemWhitespace, " "),
				mk(itemVariable, "hostName"),
				mk(itemWhitespace, " "),
				mk(itemOperator, "="),
				mk(itemWhitespace, " "),
				mk(itemExpressionOpen, "{"),
				mk(itemVariable, "host"),
				mk(itemExpressionClose, "}"),
				mk(itemWhitespace, " "),
				// .local $h = {$hostName}
				mk(itemKeyword, "local"),
				mk(itemWhitespace, " "),
				mk(itemVariable, "h"),
				mk(itemWhitespace, " "),
				mk(itemOperator, "="),
				mk(itemWhitespace, " "),
				mk(itemExpressionOpen, "{"),
				mk(itemVariable, "host"),
				mk(itemExpressionClose, "}"),
				tokenEOF,
			},
		},
		{
			name:  "input declaration",
			input: ".input {$host} .input {$user}",
			expected: []item{
				// .input {$hostName}
				mk(itemKeyword, "input"),
				mk(itemWhitespace, " "),
				mk(itemExpressionOpen, "{"),
				mk(itemVariable, "host"),
				mk(itemExpressionClose, "}"),
				mk(itemWhitespace, " "),
				// .input ${host}
				mk(itemKeyword, "input"),
				mk(itemWhitespace, " "),
				mk(itemExpressionOpen, "{"),
				mk(itemVariable, "user"),
				mk(itemExpressionClose, "}"),
				tokenEOF,
			},
		},
		{
			name:  "reserved declaration",
			input: ".output",
			expected: []item{
				mk(itemKeyword, "output"),
				tokenEOF,
			},
		},
		{
			name:  "matcher",
			input: ".match {$n} 0 {{no apples}} 1 {{{$n} apple}} * {{{$n} apples}}",
			expected: []item{
				mk(itemKeyword, "match"),
				mk(itemWhitespace, " "),
				tokenExpressionOpen,
				mk(itemVariable, "n"),
				tokenExpressionClose,
				mk(itemWhitespace, " "),
				// 0 {{no apples}}
				mk(itemLiteral, "0"),
				mk(itemWhitespace, " "),
				mk(itemQuotedPatternOpen, "{{"),
				mk(itemText, "no apples"),
				mk(itemQuotedPatternClose, "}}"),
				mk(itemWhitespace, " "),
				// 1 {{{$n} apple}}
				mk(itemLiteral, "1"),
				mk(itemWhitespace, " "),
				mk(itemQuotedPatternOpen, "{{"),
				mk(itemExpressionOpen, "{"),
				mk(itemVariable, "n"),
				mk(itemExpressionClose, "}"),
				mk(itemText, " apple"),
				mk(itemQuotedPatternClose, "}}"),
				mk(itemWhitespace, " "),
				// * {{{$n} apples}}
				mk(itemLiteral, "*"),
				mk(itemWhitespace, " "),
				mk(itemQuotedPatternOpen, "{{"),
				mk(itemExpressionOpen, "{"),
				mk(itemVariable, "n"),
				mk(itemExpressionClose, "}"),
				mk(itemText, " apples"),
				mk(itemQuotedPatternClose, "}}"),

				tokenEOF,
			},
		},
	} {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			l := lex(test.input)

			// collect items

			var items []item

			for {
				v := l.nextItem()

				items = append(items, v)
				if v.typ == itemEOF || v.typ == itemError {
					break
				}
			}

			// assert

			assert.Equal(t, test.expected, items)
		})
	}
}
