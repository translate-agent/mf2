package mf2

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_Builder(t *testing.T) {
	t.Parallel()

	//nolint:lll
	for _, test := range []struct {
		name     string
		b        *Builder
		expected string
	}{
		{
			"simple message, empty text",
			NewBuilder().Text(""),
			"",
		},
		{
			"simple message, simple text",
			NewBuilder().Text("Hello, World!"),
			"Hello, World!",
		},
		{
			"simple message, text starts with whitespace char",
			NewBuilder().Text(" hello"),
			" hello",
		},
		{
			"simple message, text with special chars",
			NewBuilder().Text("{Hello}\\, {World}!"),
			"\\{Hello\\}\\\\, \\{World\\}!",
		},
		{
			"simple message, text with literal",
			NewBuilder().
				Text("Hello, ").
				Expr(Literal("World")).
				Text("!"),
			"Hello, { World }!",
		},
		{
			"simple message, text and expr with options",
			NewBuilder().
				Text("Hello, ").
				Expr(Var("world").
					Func("upper",
						LiteralOption("limit", 2),
						VarOption("min", "min"),
						LiteralOption("type", "integer"),
						LiteralOption("x", "y z"),
						LiteralOption("host", ""))).
				Text("!"),
			"Hello, { $world :upper limit = 2 min = $min type = integer x = |y z| host = || }!",
		},
		{
			"complex message, period char",
			NewBuilder().Text("."),
			"{{.}}",
		},
		{
			"complex message, text starts with a period char",
			NewBuilder().Text(".ok"),
			"{{.ok}}",
		},
		{
			"complex message, text with special chars",
			NewBuilder().Local("var", Literal("greeting")),
			".local $var = { greeting }\n{{}}",
		},
		{
			"complex message, local declaration",
			NewBuilder().
				Local("hostName", Var("host")).
				Expr(Var("hostName")),
			".local $hostName = { $host }\n{{{ $hostName }}}",
		},
		{
			"complex message, input declaration",
			NewBuilder().
				Input(Var("host")).
				Expr(Var("host")),
			".input { $host }\n{{{ $host }}}",
		},
		{
			"complex message, input and local declaration",
			NewBuilder().
				Local("hostName", Var("host")).
				Input(Var("host")).
				Expr(Var("host")),
			".input { $host }\n.local $hostName = { $host }\n{{{ $host }}}",
		},
		{
			"complex message, matcher with multiple keys",
			NewBuilder().
				Match(
					Var("i"),
					Var("j"),
				).
				Keys(1, 2).Text("{first}").
				Keys(2, 0).Text("second ").Expr(Var("i")).
				Keys(3, 0).Expr(Literal("\\a|")).
				Keys("*", "*").Expr(Literal(1)),
			".match { $i } { $j }\n1 2 {{\\{first\\}}}\n2 0 {{second { $i }}}\n3 0 {{{ |\\\\a\\|| }}}\n* * {{{ 1 }}}",
		},
		{
			"complex message, matcher with multiple keys and local declarations",
			NewBuilder().
				Input(Var("i")).
				Local("hostName", Var("i")).
				Match(
					Var("i"),
					Var("j"),
				).
				Keys(1, 2).Text("{first}").
				Keys(2, 0).Text("second ").Expr(Var("i")).
				Keys(3, 0).Expr(Literal("\\a|")).
				Keys("*", "*").Expr(Literal(1)),
			".input { $i }\n.local $hostName = { $i }\n.match { $i } { $j }\n1 2 {{\\{first\\}}}\n2 0 {{second { $i }}}\n3 0 {{{ |\\\\a\\|| }}}\n* * {{{ 1 }}}",
		},
		{
			"spacing",
			NewBuilder().
				Match(
					Var("i"),
					Var("j"),
				).
				Keys(1, 2).Text("{first}").
				Keys(2, 0).Text("second ").Expr(Var("i")).
				Keys(3, 0).Expr(Literal("\\a|")).
				Keys("*", "*").Expr(Literal(1)).
				Spacing(""),
			".match{$i}{$j}\n1 2{{\\{first\\}}}\n2 0{{second {$i}}}\n3 0{{{|\\\\a\\||}}}\n* *{{{1}}}",
		},
		{
			"attributes",
			NewBuilder().
				Text("Attributes for variable expression ").
				Expr(
					Var("i").
						Attr(
							VarAttribute("attr1", "var"),
							LiteralAttribute("attr2", "literal"),
							EmptyAttribute("empty"),
						),
				),
			"Attributes for variable expression { $i @attr1 = $var @attr2 = literal @empty }",
		},
		{
			"markup",
			NewBuilder().
				OpenMarkup(
					"open",
					LiteralOption("opt1", "val1"),
					LiteralAttribute("attr1", 1),
					VarOption("opt2", "var"),
				).
				Text(" something ").
				CloseMarkup(
					"close",
					EmptyAttribute("empty1"),
					VarAttribute("attr1", "var"),
				).
				SelfCloseMarkup(
					"selfClosing",
					LiteralAttribute("attr1", "༼ つ ◕_◕ ༽つ"),
				),
			"{ #open opt1 = val1 opt2 = $var @attr1 = 1 } something { /close @empty1 @attr1 = $var }{ #selfClosing @attr1 = |༼ つ ◕_◕ ༽つ| / }",
		},
	} {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			actual, err := test.b.Build()
			require.NoError(t, err)

			require.Equal(t, test.expected, actual)
		})
	}
}
