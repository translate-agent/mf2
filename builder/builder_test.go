package builder

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_Builder(t *testing.T) {
	t.Parallel()

	//nolint:govet, lll
	for _, test := range []struct {
		name, expected string
		b              *Builder
	}{
		{
			"simple message, empty text",
			"",
			New().Text(""),
		},
		{
			"simple message, simple text",
			"Hello, World!",
			New().Text("Hello, World!"),
		},
		{
			"simple message, text starts with whitespace char",
			" hello",
			New().Text(" hello"),
		},
		{
			"simple message, text with special chars",
			"\\{Hello\\}\\\\, \\{World\\}!",
			New().Text("{Hello}\\, {World}!"),
		},
		{
			"simple message, text with literal",
			"Hello, { |World| }!",
			New().
				Text("Hello, ").
				Expr(Literal("World")).
				Text("!"),
		},
		{
			"simple message, text and expr with options",
			"Hello, { $world :upper limit = 2 min = $min type = |integer| }!",
			New().
				Text("Hello, ").
				Expr(Var("$world").Func(":upper", Option("limit", 2), Option("min", "$min"), Option("type", "integer"))).
				Text("!"),
		},
		{
			"simple message, text with markup-like function",
			"Hello { +link } World { -link }",
			New().
				Text("Hello ").
				Expr(Func("+link")).
				Text(" World ").
				Expr(Func("-link")),
		},
		{
			"complex message, period char",
			"{{.}}",
			New().Text("."),
		},
		{
			"complex message, text starts with a period char",
			"{{.ok}}",
			New().Text(".ok"),
		},
		{
			"complex message, text with special chars",
			".local $var = { |greeting| }\n{{}}",
			New().Local("$var", Literal("greeting")),
		},
		{
			"complex message, local declaration",
			".local $hostName = { $host }\n{{{ $hostName }}}",
			New().
				Local("$hostName", Var("$host")).
				Expr(Var("$hostName")),
		},
		{
			"complex message, input declaration",
			".input { $host }\n{{{ $host }}}",
			New().
				Input(Var("$host")).
				Expr(Var("$host")),
		},
		{
			"complex message, input and local declaration",
			".input { $host }\n.local $hostName = { $host }\n{{{ $host }}}",
			New().
				Local("$hostName", Var("$host")).
				Input(Var("$host")).
				Expr(Var("$host")),
		},
		{
			"complex message, matcher with multiple keys",
			".match {$i} {$j}\n1 2 {{\\{first\\}}}\n2 0 {{second { $i }}}\n3 0 {{{ |\\\\a\\|| }}}\n* * {{{ 1 }}}",
			New().
				Match(
					Var("$i"),
					Var("$j"),
				).
				Key(1, 2).Text("{first}").
				Key(2, 0).Text("second").Expr(Var("$i")).
				Key(3, 0).Expr(Literal("\\a|")).
				Key("*", "*").Expr(Literal(1)),
		},
		{
			"complex message, matcher with multiple keys and local declarations",
			".input { $i }\n.local $hostName = { $i }\n.match {$i} {$j}\n1 2 {{\\{first\\}}}\n2 0 {{second { $i }}}\n3 0 {{{ |\\\\a\\|| }}}\n* * {{{ 1 }}}",
			New().
				Input(Var("$i")).
				Local("$hostName", Var("$i")).
				Match(
					Var("$i"),
					Var("$j"),
				).
				Key(1, 2).Text("{first}").
				Key(2, 0).Text("second").Expr(Var("$i")).
				Key(3, 0).Expr(Literal("\\a|")).
				Key("*", "*").Expr(Literal(1)),
		},
	} {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, test.expected, test.b.MustBuild())
		})
	}
}
