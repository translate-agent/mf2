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
			input: "{:rand seed=1 log:level=$log}",
			expected: []item{
				mk(itemExpressionOpen, "{"),
				mk(itemFunction, ":rand"),
				mk(itemWhitespace, " "),
				mk(itemOption, "seed"),
				mk(itemOperator, "="),
				mk(itemNumberLiteral, "1"),
				mk(itemWhitespace, " "),
				mk(itemOption, "log:level"),
				mk(itemOperator, "="),
				mk(itemVariable, "log"),
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
				mk(itemVariable, "count"),
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
				mk(itemVariable, "guest"),
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
				mk(itemQuotedLiteral, ""),
				mk(itemExpressionClose, "}"),
				mk(itemEOF, ""),
			},
		},
		{
			name:  "quoted literal",
			input: "{|Hello, world!| :uppercase}",
			expected: []item{
				mk(itemExpressionOpen, "{"),
				mk(itemQuotedLiteral, "Hello, world!"),
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
				mk(itemNumberLiteral, "-1.9e+10"),
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
				mk(itemUnquotedLiteral, "hello"),
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
				mk(itemQuotedLiteral, "hello"),
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
				mk(itemLocalKeyword, ".local"),
				mk(itemWhitespace, " "),
				mk(itemVariable, "hostName"),
				mk(itemWhitespace, " "),
				mk(itemOperator, "="),
				mk(itemWhitespace, " "),
				mk(itemExpressionOpen, "{"),
				mk(itemVariable, "host"),
				mk(itemExpressionClose, "}"),
				mk(itemWhitespace, " "),
				// .local $h = {$host}
				mk(itemLocalKeyword, ".local"),
				mk(itemWhitespace, " "),
				mk(itemVariable, "h"),
				mk(itemWhitespace, " "),
				mk(itemOperator, "="),
				mk(itemWhitespace, " "),
				mk(itemExpressionOpen, "{"),
				mk(itemVariable, "host"),
				mk(itemExpressionClose, "}"),
				mk(itemEOF, ""),
			},
		},
		{
			name:  "input declaration",
			input: ".input {$host} .input {$user}",
			expected: []item{
				// .input {$host}
				mk(itemInputKeyword, ".input"),
				mk(itemWhitespace, " "),
				mk(itemExpressionOpen, "{"),
				mk(itemVariable, "host"),
				mk(itemExpressionClose, "}"),
				mk(itemWhitespace, " "),
				// .input ${user}
				mk(itemInputKeyword, ".input"),
				mk(itemWhitespace, " "),
				mk(itemExpressionOpen, "{"),
				mk(itemVariable, "user"),
				mk(itemExpressionClose, "}"),
				mk(itemEOF, ""),
			},
		},
		{
			name:  "reserved declaration",
			input: ".output",
			expected: []item{
				mk(itemReservedKeyword, "output"),
				mk(itemEOF, ""),
			},
		},
		{
			name:  "matcher",
			input: ".match {$n} 0 {{no apples}} 1 {{{$n} apple}} * {{{$n} apples}}",
			expected: []item{
				mk(itemMatchKeyword, ".match"),
				mk(itemWhitespace, " "),
				mk(itemExpressionOpen, "{"),
				mk(itemVariable, "n"),
				mk(itemExpressionClose, "}"),
				mk(itemWhitespace, " "),
				// 0 {{no apples}}
				mk(itemNumberLiteral, "0"),
				mk(itemWhitespace, " "),
				mk(itemQuotedPatternOpen, "{{"),
				mk(itemText, "no apples"),
				mk(itemQuotedPatternClose, "}}"),
				mk(itemWhitespace, " "),
				// 1 {{{$n} apple}}
				mk(itemNumberLiteral, "1"),
				mk(itemWhitespace, " "),
				mk(itemQuotedPatternOpen, "{{"),
				mk(itemExpressionOpen, "{"),
				mk(itemVariable, "n"),
				mk(itemExpressionClose, "}"),
				mk(itemText, " apple"),
				mk(itemQuotedPatternClose, "}}"),
				mk(itemWhitespace, " "),
				// * {{{$n} apples}}
				mk(itemCatchAllKey, "*"),
				mk(itemWhitespace, " "),
				mk(itemQuotedPatternOpen, "{{"),
				mk(itemExpressionOpen, "{"),
				mk(itemVariable, "n"),
				mk(itemExpressionClose, "}"),
				mk(itemText, " apples"),
				mk(itemQuotedPatternClose, "}}"),

				mk(itemEOF, ""),
			},
		},
		{
			name:  "complex message without declaration",
			input: "{{Hello, {|literal|} World!}}",
			expected: []item{
				mk(itemQuotedPatternOpen, "{{"),
				mk(itemText, "Hello, "),
				mk(itemExpressionOpen, "{"),
				mk(itemQuotedLiteral, "literal"),
				mk(itemExpressionClose, "}"),
				mk(itemText, " World!"),
				mk(itemQuotedPatternClose, "}}"),
			},
		},
	} {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			assertItems(t, test.expected, lex(test.input))
		})
	}
}

func assertItems(t *testing.T, expected []item, l *lexer) {
	t.Helper()

	logItems := make([]func(), 0, len(expected))

	for _, exp := range expected {
		v := l.nextItem()

		logItems = append(logItems, logItem(t, exp, *l))

		if !assert.Equal(t, exp, v) {
			for _, f := range logItems {
				f()
			}
		}
	}
}

func logItem(t *testing.T, expected item, l lexer) func() {
	t.Helper()

	return func() {
		f := func(b bool) string {
			if b {
				return "✓"
			}

			return " "
		}

		t.Logf("c%s p%s e%s %-60s e%s(%s) a%s(%s)\n",
			f(l.isComplexMessage), f(l.isPattern), f(l.isExpression),
			"'"+l.input[l.pos:]+"'", "'"+expected.val+"'", expected.typ, "'"+l.item.val+"'", l.item.typ)
	}
}
