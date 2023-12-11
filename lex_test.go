package mf2

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_lex(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name     string
		input    string // MessageFormat2 formatted string
		expected []item
	}{
		{
			name:     "empty simple message",
			input:    "",
			expected: []item{mk(itemEOF, "")},
		},
		{
			name:  "text",
			input: `escaped text: \\ \} \{`,
			expected: []item{
				mk(itemText, `escaped text: \ } {`),
				mk(itemEOF, ""),
			},
		},
		{
			name:  "function",
			input: "{:rand}",
			expected: []item{
				mk(itemExpressionOpen, "{"),
				mk(itemFunction, ":rand"),
				mk(itemExpressionClose, "}"),
				mk(itemEOF, ""),
			},
		},

		{
			name:  "opening function",
			input: "{+button}",
			expected: []item{
				mk(itemExpressionOpen, "{"),
				mk(itemFunction, "+button"),
				mk(itemExpressionClose, "}"),
				mk(itemEOF, ""),
			},
		},
		{
			name:  "closing function",
			input: "{-button}",
			expected: []item{
				mk(itemExpressionOpen, "{"),
				mk(itemFunction, "-button"),
				mk(itemExpressionClose, "}"),
				mk(itemEOF, ""),
			},
		},
		{
			name:  "opening and closing functions",
			input: "{+button}Submit{-button}",
			expected: []item{
				mk(itemExpressionOpen, "{"),
				mk(itemFunction, "+button"),
				mk(itemExpressionClose, "}"),
				mk(itemText, "Submit"),
				mk(itemExpressionOpen, "{"),
				mk(itemFunction, "-button"),
				mk(itemExpressionClose, "}"),
				mk(itemEOF, ""),
			},
		},
		{
			name:  "variable",
			input: "{$count :math:round}",
			expected: []item{
				mk(itemExpressionOpen, "{"),
				mk(itemVariable, "$count"),
				mk(itemWhitespace, " "),
				mk(itemFunction, ":math:round"),
				mk(itemExpressionClose, "}"),
				mk(itemEOF, ""),
			},
		},
		{
			name:  "text with variable",
			input: "Hello, {$guest}!",
			expected: []item{
				mk(itemText, "Hello, "),
				mk(itemExpressionOpen, "{"),
				mk(itemVariable, "$guest"),
				mk(itemExpressionClose, "}"),
				mk(itemText, "!"),
				mk(itemEOF, ""),
			},
		},
		{
			name:  "empty quoted literal",
			input: "{||}",
			expected: []item{
				mk(itemExpressionOpen, "{"),
				mk(itemLiteral, ""),
				mk(itemExpressionClose, "}"),
				mk(itemEOF, ""),
			},
		},
		{
			name:  "quoted literal",
			input: "{|Hello, world!| :uppercase}",
			expected: []item{
				mk(itemExpressionOpen, "{"),
				mk(itemLiteral, "Hello, world!"),
				mk(itemWhitespace, " "),
				mk(itemFunction, ":uppercase"),
				mk(itemExpressionClose, "}"),
				mk(itemEOF, ""),
			},
		},
		{
			name:  "number literal",
			input: "{-1.9e+10 :odd}",
			expected: []item{
				mk(itemExpressionOpen, "{"),
				mk(itemLiteral, "-1.9e+10"),
				mk(itemWhitespace, " "),
				mk(itemFunction, ":odd"),
				mk(itemExpressionClose, "}"),
				mk(itemEOF, ""),
			},
		},
		{
			name:  "unquoted literal",
			input: "{hello :uppercase}",
			expected: []item{
				mk(itemExpressionOpen, "{"),
				mk(itemLiteral, "hello"),
				mk(itemWhitespace, " "),
				mk(itemFunction, ":uppercase"),
				mk(itemExpressionClose, "}"),
				mk(itemEOF, ""),
			},
		},
		{
			name:  "reserved",
			input: `{!a @b #c %d *e <|hello| > /\{ ?\| ~\}}`,
			expected: []item{
				mk(itemExpressionOpen, "{"),
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
				mk(itemExpressionClose, "}"),
				mk(itemEOF, ""),
			},
		},
		{
			name:  "private use", // TODO: incomplete
			input: "{^ &}",
			expected: []item{
				mk(itemExpressionOpen, "{"),
				mk(itemPrivate, "^"),
				mk(itemWhitespace, " "),
				mk(itemPrivate, "&"),
				mk(itemExpressionClose, "}"),
				mk(itemEOF, ""),
			},
		},
		{
			name:  "local declaration",
			input: ".local $hostName = {$host} .local $h = {$host}",
			expected: []item{
				// .local $hostName = {$host}
				mk(itemKeyword, ".local"),
				mk(itemWhitespace, " "),
				mk(itemVariable, "$hostName"),
				mk(itemWhitespace, " "),
				mk(itemOperator, "="),
				mk(itemWhitespace, " "),
				mk(itemExpressionOpen, "{"),
				mk(itemVariable, "$host"),
				mk(itemExpressionClose, "}"),
				mk(itemWhitespace, " "),
				// .local $h = {$host}
				mk(itemKeyword, ".local"),
				mk(itemWhitespace, " "),
				mk(itemVariable, "$h"),
				mk(itemWhitespace, " "),
				mk(itemOperator, "="),
				mk(itemWhitespace, " "),
				mk(itemExpressionOpen, "{"),
				mk(itemVariable, "$host"),
				mk(itemExpressionClose, "}"),
				mk(itemEOF, ""),
			},
		},
		{
			name:  "input declaration",
			input: ".input {$host} .input {$user}",
			expected: []item{
				// .input {$host}
				mk(itemKeyword, ".input"),
				mk(itemWhitespace, " "),
				mk(itemExpressionOpen, "{"),
				mk(itemVariable, "$host"),
				mk(itemExpressionClose, "}"),
				mk(itemWhitespace, " "),
				// .input ${user}
				mk(itemKeyword, ".input"),
				mk(itemWhitespace, " "),
				mk(itemExpressionOpen, "{"),
				mk(itemVariable, "$user"),
				mk(itemExpressionClose, "}"),
				mk(itemEOF, ""),
			},
		},
		{
			name:  "reserved declaration",
			input: ".output",
			expected: []item{
				mk(itemKeyword, ".output"),
				mk(itemEOF, ""),
			},
		},
		{
			name:  "matcher",
			input: ".match {$n} 0 {{no apples}} 1 {{{$n} apple}} * {{{$n} apples}}",
			expected: []item{
				mk(itemKeyword, ".match"),
				mk(itemWhitespace, " "),
				mk(itemExpressionOpen, "{"),
				mk(itemVariable, "$n"),
				mk(itemExpressionClose, "}"),
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
				mk(itemVariable, "$n"),
				mk(itemExpressionClose, "}"),
				mk(itemText, " apple"),
				mk(itemQuotedPatternClose, "}}"),
				mk(itemWhitespace, " "),
				// * {{{$n} apples}}
				mk(itemLiteral, "*"),
				mk(itemWhitespace, " "),
				mk(itemQuotedPatternOpen, "{{"),
				mk(itemExpressionOpen, "{"),
				mk(itemVariable, "$n"),
				mk(itemExpressionClose, "}"),
				mk(itemText, " apples"),
				mk(itemQuotedPatternClose, "}}"),

				mk(itemEOF, ""),
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

				logNext(t, l, v)

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

func logNext(t *testing.T, l *lexer, i item) {
	t.Helper()

	f := func(b bool) string {
		if b {
			return "✓"
		}

		return " "
	}

	t.Logf("c%s p%s e%s %-50s %-12s %s\n",
		f(l.isComplexMessage), f(l.isPattern), f(l.isExpression),
		"'"+l.input[l.pos:]+"'", "'"+i.val+"'", i.typ)
}
